package lipc

import "github.com/godbus/dbus/v5"

func NameForHeaderField(t dbus.HeaderField) string {
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

func NameForLipcError(errnum uint32) string {
	switch errnum {
	case 0:
		return "lipcErrNone"
	case 1:
		return "lipcErrUnknown"
	case 2:
		return "lipcErrInternal"
	case 3:
		return "lipcErrNoSuchSource"
	case 4:
		return "lipcErrOperationNotSupported"
	case 5:
		return "lipcErrOutOfMemory"
	case 6:
		return "lipcErrSubscriptionFailed"
	case 7:
		return "lipcErrNoSuchParam"
	case 8:
		return "lipcErrNoSuchProperty"
	case 9:
		return "lipcErrAccessNotAllowed"
	case 0xa:
		return "lipcErrBufferTooSmall"
	case 0xb:
		return "lipcErrInvalidHandle"
	case 0xc:
		return "lipcErrInvalidArg"
	case 0xd:
		return "lipcErrOperationNotAllowed"
	case 0xe:
		return "lipcErrParamsSizeExceeded"
	case 0xf:
		return "lipcErrTimedOut"
	case 0x10:
		return "lipcErrServiceNameTooLong"
	case 0x11:
		return "lipcErrDuplicateServiceName"
	case 0x12:
		return "lipcErrInitDBus"
	case 0x100:
		return "lipcPropErrInvalidState"
	case 0x101:
		return "lipcPropErrNotInitialized"
	case 0x102:
		return "lipcPropErrInternal"
	default:
		return "Unknown"
	}
}
