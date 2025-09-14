// colmi holds a few functions for interacting with a smart ring
// https://colmi.puxtril.com/
package colmi

import (
	"fmt"

	"github.com/google/uuid"
)

// See https://colmi.puxtril.com/ for some documentation of the two characteristics.
var R02_BIG_DATA_SERVICE_UUID = uuid.MustParse("de5bf728-d711-4e47-af26-65e3012a5dc7")
var R02_BIG_DATA_WRITE_UUID = uuid.MustParse("de5bf72a-d711-4e47-af26-65e3012a5dc7")
var R02_BIG_DATA_READ_UUID = uuid.MustParse("de5bf729-d711-4e47-af26-65e3012a5dc7")

var R02_COMMANDS_SERVICE_UUID = uuid.MustParse("6e40fff0-b5a3-f393-e0a9-e50e24dcca9e")
var R02_COMMANDS_WRITE_UUID = uuid.MustParse("6e400002-b5a3-f393-e0a9-e50e24dcca9e")
var R02_COMMANDS_READ_UUID = uuid.MustParse("6e400003-b5a3-f393-e0a9-e50e24dcca9e")

// These aren't Colmi specific
var DEVICE_INFO_UUID = uuid.MustParse("0000180A-0000-1000-8000-00805F9B34FB")
var DEVICE_HW_UUID = uuid.MustParse("00002A27-0000-1000-8000-00805F9B34FB")
var DEVICE_FW_UUID = uuid.MustParse("00002A26-0000-1000-8000-00805F9B34FB")

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
	ACTION_INTO_CAMERA_UI
	ACTION_TAKE_PHOTO
	ACTION_FINISH
	ACTION_ENABLE_CAMERA_GESTURE
	ACTION_KEEP_SCREEN_ON
	ACTION_DISABLE_CAMERA_GESTURE
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
	if data[1] != byte(ACTION_TAKE_PHOTO) {
		return false
	}
	if data[15] != 0x02+byte(ACTION_TAKE_PHOTO) {
		return false
	}
	for i := 2; i < 15; i++ {
		if data[i] != 0x00 {
			return false
		}
	}
	return true
}
