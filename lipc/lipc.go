// Package lipc provides a cgo-less interface for getting and setting LIPC properties over DBus.
// Hasharrays are not supported, as they're a more complex interface involving shared memory.
package lipc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/godbus/dbus/v5"
)

type PropType string

const (
	IntProp PropType = "Int"
	StrProp PropType = "Str"
)

type PropMessageType string

const (
	GetProp PropMessageType = "get"
	SetProp PropMessageType = "set"
)

// makePropertyMessage constructs a D-Bus message for getting or setting an LIPC property.
func makePropertyMessage[T any](msgType PropMessageType, service, property string, propType PropType, body ...T) (*dbus.Message, error) {
	msg := new(dbus.Message)
	msg.Type = dbus.TypeMethodCall
	msg.Flags = (dbus.FlagNoAutoStart)
	msg.Headers = make(map[dbus.HeaderField]dbus.Variant)

	// The path is always /default
	msg.Headers[dbus.FieldPath] = dbus.MakeVariant(dbus.ObjectPath("/default"))

	// do we need destination and interface?
	// I think LIPC sets both, but strictly speaking the interface may not be required?
	msg.Headers[dbus.FieldDestination] = dbus.MakeVariant(service)
	msg.Headers[dbus.FieldInterface] = dbus.MakeVariant(service)

	// "getflIntensityInt", "setflIntensityInt"
	// "getstatusStr", "setstatusStr"
	method := fmt.Sprintf("%s%s%s", msgType, property, propType)
	msg.Headers[dbus.FieldMember] = dbus.MakeVariant(method)

	// LIPC seems to set the _calling_ service name in the body for gets, but that's optional
	// since not all connections have a service name anyway.
	if msgType == SetProp {
		var interfaceBody []any
		for _, v := range body {
			interfaceBody = append(interfaceBody, v)
		}
		msg.Body = interfaceBody
		if len(interfaceBody) > 0 {
			msg.Headers[dbus.FieldSignature] = dbus.MakeVariant(dbus.SignatureOf(interfaceBody...))
		}
	}

	err := msg.IsValid()
	if err != nil {
		slog.Error("message is not valid", "error", err)
		return nil, err
	}
	return msg, nil
}

func GetProperty[T any](ctx context.Context, conn *dbus.Conn, service, property string) (T, error) {
	return doProperty[T](ctx, conn, service, property, GetProp)
}

func SetProperty[T any](ctx context.Context, conn *dbus.Conn, service, property string, value T) error {
	_, err := doProperty(ctx, conn, service, property, SetProp, value)
	return err
}

func doProperty[T any](ctx context.Context, conn *dbus.Conn, service, property string, msgType PropMessageType, value ...T) (T, error) {
	var propType PropType
	switch any(*new(T)).(type) {
	case string:
		propType = StrProp
	case int32:
		propType = IntProp
	default:
		return *new(T), fmt.Errorf("unsupported property type %T", any(value))
	}
	message, err := makePropertyMessage(msgType, service, property, propType, value...)
	if err != nil {
		return *new(T), err
	}

	call := <-conn.SendWithContext(ctx, message, make(chan *dbus.Call, 1)).Done
	if call.Err != nil {
		slog.Error("failed to get property", "error", call.Err)
		return *new(T), call.Err
	}
	slog.Debug("got property response", "body", call.Body)

	var propValue T
	var status uint32
	if msgType == GetProp {
		err := call.Store(&status, &propValue)
		if err != nil {
			slog.Error("failed to store call body", "error", err)
			return *new(T), err
		}
	} else {
		err := call.Store(&status)
		if err != nil {
			slog.Error("failed to store call body", "error", err)
			return *new(T), err
		}
	}
	if status != 0 {
		return *new(T), fmt.Errorf("non-zero status %d from %sStrProperty", status, msgType)
	}
	return propValue, nil
}
