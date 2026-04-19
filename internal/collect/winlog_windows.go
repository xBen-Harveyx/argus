//go:build windows

package collect

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/ben/argus/internal/model"
)

var (
	wevtapiDLL                  = syscall.NewLazyDLL("wevtapi.dll")
	evtQueryProc                = wevtapiDLL.NewProc("EvtQuery")
	evtNextProc                 = wevtapiDLL.NewProc("EvtNext")
	evtRenderProc               = wevtapiDLL.NewProc("EvtRender")
	evtCloseProc                = wevtapiDLL.NewProc("EvtClose")
	evtOpenPublisherMetadataProc = wevtapiDLL.NewProc("EvtOpenPublisherMetadata")
	evtFormatMessageProc        = wevtapiDLL.NewProc("EvtFormatMessage")
)

const (
	evtQueryChannelPath    = 0x1
	evtQueryForward        = 0x100
	evtRenderEventXML      = 1
	evtFormatMessageEvent  = 1
	errorInsufficientBuffer = 122
	errorNoMoreItems       = 259
)

type windowsEventCollector struct{}

func NewEventCollector() EventCollector {
	return windowsEventCollector{}
}

func (windowsEventCollector) Collect(ctx context.Context, start, end time.Time) ([]model.EventRecord, []string) {
	type source struct {
		channel  string
		provider string
	}
	sources := []source{
		{channel: "Microsoft-Windows-WLAN-AutoConfig/Operational", provider: "Microsoft-Windows-WLAN-AutoConfig"},
		{channel: "Microsoft-Windows-Dhcp-Client/Operational", provider: "Microsoft-Windows-Dhcp-Client"},
		{channel: "Microsoft-Windows-DNS-Client/Operational", provider: "Microsoft-Windows-DNS-Client"},
		{channel: "Microsoft-Windows-TCPIP/Operational", provider: "Microsoft-Windows-TCPIP"},
		{channel: "Microsoft-Windows-NetworkProfile/Operational", provider: "Microsoft-Windows-NetworkProfile"},
	}

	var (
		events   []model.EventRecord
		warnings []string
	)
	for _, src := range sources {
		select {
		case <-ctx.Done():
			return events, append(warnings, ctx.Err().Error())
		default:
		}

		records, err := queryChannel(src.channel, src.provider, start, end)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("event query failed for %s: %v", src.channel, err))
			continue
		}
		events = append(events, records...)
	}
	return events, warnings
}

func queryChannel(channel, provider string, start, end time.Time) ([]model.EventRecord, error) {
	query := fmt.Sprintf("*[System[TimeCreated[@SystemTime>='%s' and @SystemTime<='%s']]]", formatEventTime(start), formatEventTime(end))
	channelPtr, _ := syscall.UTF16PtrFromString(channel)
	queryPtr, _ := syscall.UTF16PtrFromString(query)
	handle, _, err := evtQueryProc.Call(0, uintptr(unsafe.Pointer(channelPtr)), uintptr(unsafe.Pointer(queryPtr)), evtQueryChannelPath|evtQueryForward)
	if handle == 0 {
		return nil, err
	}
	defer evtCloseProc.Call(handle)

	metadataHandle, _ := openPublisherMetadata(provider)
	if metadataHandle != 0 {
		defer evtCloseProc.Call(metadataHandle)
	}

	var results []model.EventRecord
	events := make([]uintptr, 16)
	for {
		var returned uint32
		ret, _, nextErr := evtNextProc.Call(
			handle,
			uintptr(len(events)),
			uintptr(unsafe.Pointer(&events[0])),
			0,
			0,
			uintptr(unsafe.Pointer(&returned)),
		)
		if ret == 0 {
			if errno(nextErr) == errorNoMoreItems {
				break
			}
			return results, nextErr
		}

		for i := uint32(0); i < returned; i++ {
			eventHandle := events[i]
			record, err := renderEvent(eventHandle, metadataHandle)
			evtCloseProc.Call(eventHandle)
			if err != nil {
				continue
			}
			results = append(results, record)
		}
	}
	return results, nil
}

func openPublisherMetadata(provider string) (uintptr, error) {
	providerPtr, _ := syscall.UTF16PtrFromString(provider)
	handle, _, err := evtOpenPublisherMetadataProc.Call(0, uintptr(unsafe.Pointer(providerPtr)), 0, 0, 0)
	if handle == 0 {
		return 0, err
	}
	return handle, nil
}

func renderEvent(eventHandle uintptr, metadataHandle uintptr) (model.EventRecord, error) {
	xmlText, err := renderXML(eventHandle)
	if err != nil {
		return model.EventRecord{}, err
	}

	var parsed eventXML
	if err := xml.Unmarshal([]byte(xmlText), &parsed); err != nil {
		return model.EventRecord{}, err
	}

	message, _ := formatMessage(metadataHandle, eventHandle)
	if message == "" {
		message = fallbackMessage(parsed.EventData)
	}

	return model.EventRecord{
		Timestamp: parsed.System.TimeCreated.SystemTime,
		Provider:  parsed.System.Provider.Name,
		EventID:   parsed.System.EventID,
		Level:     levelString(parsed.System.Level),
		Channel:   parsed.System.Channel,
		Message:   message,
	}, nil
}

func renderXML(eventHandle uintptr) (string, error) {
	var bufferUsed uint32
	ret, _, err := evtRenderProc.Call(0, eventHandle, evtRenderEventXML, 0, 0, uintptr(unsafe.Pointer(&bufferUsed)), 0)
	if ret == 0 && errno(err) != errorInsufficientBuffer {
		return "", err
	}

	buffer := make([]uint16, bufferUsed/2+1)
	ret, _, err = evtRenderProc.Call(0, eventHandle, evtRenderEventXML, uintptr(bufferUsed), uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&bufferUsed)), 0)
	if ret == 0 {
		return "", err
	}
	return syscall.UTF16ToString(buffer), nil
}

func formatMessage(metadataHandle uintptr, eventHandle uintptr) (string, error) {
	if metadataHandle == 0 {
		return "", nil
	}
	var bufferUsed uint32
	ret, _, err := evtFormatMessageProc.Call(metadataHandle, eventHandle, 0, 0, 0, evtFormatMessageEvent, 0, 0, uintptr(unsafe.Pointer(&bufferUsed)))
	if ret == 0 && errno(err) != errorInsufficientBuffer {
		return "", err
	}
	buffer := make([]uint16, bufferUsed+1)
	ret, _, err = evtFormatMessageProc.Call(metadataHandle, eventHandle, 0, 0, 0, evtFormatMessageEvent, uintptr(bufferUsed), uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&bufferUsed)))
	if ret == 0 {
		return "", err
	}
	return strings.TrimSpace(syscall.UTF16ToString(buffer)), nil
}

func formatEventTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

func levelString(level int) string {
	switch level {
	case 1:
		return "critical"
	case 2:
		return "error"
	case 3:
		return "warning"
	case 4:
		return "information"
	case 5:
		return "verbose"
	default:
		return "unknown"
	}
}

func fallbackMessage(data []eventDataItem) string {
	parts := make([]string, 0, len(data))
	for _, item := range data {
		if strings.TrimSpace(item.Value) == "" {
			continue
		}
		if item.Name != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", item.Name, item.Value))
		} else {
			parts = append(parts, item.Value)
		}
	}
	return strings.Join(parts, "; ")
}

func errno(err error) uintptr {
	if err == nil {
		return 0
	}
	if errno, ok := err.(syscall.Errno); ok {
		return uintptr(errno)
	}
	return 0
}

type eventXML struct {
	System struct {
		Provider struct {
			Name string `xml:"Name,attr"`
		} `xml:"Provider"`
		EventID int `xml:"EventID"`
		Level   int `xml:"Level"`
		Channel string `xml:"Channel"`
		TimeCreated struct {
			SystemTime string `xml:"SystemTime,attr"`
		} `xml:"TimeCreated"`
	} `xml:"System"`
	EventData []eventDataItem `xml:"EventData>Data"`
}

type eventDataItem struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:",chardata"`
}
