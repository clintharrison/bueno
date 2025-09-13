package lipc

import "github.com/godbus/dbus/v5"

// this is useful for debugging...
func nameForHeaderField(t dbus.HeaderField) string {
	switch t {
	case dbus.FieldPath:
		return "Path"
	case dbus.FieldInterface:
		return "Interface"
	case dbus.FieldMember:
		return "Member"
	case dbus.FieldErrorName:
		return "ErrorName"
	case dbus.FieldReplySerial:
		return "ReplySerial"
	case dbus.FieldDestination:
		return "Destination"
	case dbus.FieldSender:
		return "Sender"
	case dbus.FieldSignature:
		return "Signature"
	case dbus.FieldUnixFDs:
		return "UnixFDs"
	}
	return "Unknown"
}

// TODO: LIPC status to error string (LipcGetErrorString in liblipc)
