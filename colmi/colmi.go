// Package colmi holds a few functions for interacting with a smart ring
// https://colmi.puxtril.com/
package colmi

import (
	"fmt"

	"github.com/google/uuid"
)

// See https://colmi.puxtril.com/ for some documentation of the two characteristics.

var BigDataServiceUUID = uuid.MustParse("de5bf728-d711-4e47-af26-65e3012a5dc7")
var BigDataWriteUUID = uuid.MustParse("de5bf72a-d711-4e47-af26-65e3012a5dc7")
var BigDataReadUUID = uuid.MustParse("de5bf729-d711-4e47-af26-65e3012a5dc7")

var CommandServiceUUID = uuid.MustParse("6e40fff0-b5a3-f393-e0a9-e50e24dcca9e")
var CommandWriteUUID = uuid.MustParse("6e400002-b5a3-f393-e0a9-e50e24dcca9e")
var CommandReadUUID = uuid.MustParse("6e400003-b5a3-f393-e0a9-e50e24dcca9e")

// DeviceInfoServiceUUID is the well-known Device Information service
var DeviceInfoServiceUUID = uuid.MustParse("0000180A-0000-1000-8000-00805F9B34FB")
var DeviceInfoHardwareUUID = uuid.MustParse("00002A27-0000-1000-8000-00805F9B34FB")
var DeviceInfoFirmwareUUID = uuid.MustParse("00002A26-0000-1000-8000-00805F9B34FB")

func MakePacket(command byte, subdata []byte) ([]byte, error) {
	packet := [16]byte{}
	packet[0] = command
	if len(subdata) > 15 {
		return nil, fmt.Errorf("subdata too long: %d bytes (max 15)", len(subdata))
	}
	for i, d := range subdata {
		packet[i+1] = d
	}
	sum := int32(0)
	for _, b := range packet[:15] {
		sum += int32(b)
	}
	packet[15] = (byte)(sum & 0xFF)
	return packet[:], nil
}

func MakeBlinkTwicePacket() ([]byte, error) {
	// Blink twice is command ID 15: https://colmi.puxtril.com/commands/#blink-twice
	return MakePacket(15, []byte{})
}

type CameraAction int

// All of these are from https://colmi.puxtril.com/commands/#camera
// but I've only tested enable/disable and take photo.
const (
	_ CameraAction = iota
	ActionIntoCameraUI
	ActionTakePhoto
	ActionFinish
	ActionEnableCameraGesture
	ActionKeepScreenOn
	ActionDisableCameraGesture
)

func MakeCameraPacket(action CameraAction) ([]byte, error) {
	// Camera requests are command ID 2: https://colmi.puxtril.com/commands/#camera
	return MakePacket(2, []byte{byte(action)})
}

func IsCameraTakePhotoAction(data []byte) bool {
	// TODO: factor out the validation and checksum from the camera constants
	if len(data) != 16 {
		return false
	}
	if data[0] != 0x02 {
		return false
	}
	if data[1] != byte(ActionTakePhoto) {
		return false
	}
	if data[15] != 0x02+byte(ActionTakePhoto) {
		return false
	}
	for i := 2; i < 15; i++ {
		if data[i] != 0x00 {
			return false
		}
	}
	return true
}
