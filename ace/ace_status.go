package ace

//#include "ace.go.h"
import "C"

import (
	"fmt"
	"log/slog"
)

var statusNames = map[int32]string{
	C.ACE_STATUS_OK:                                 "ACE_STATUS_OK",
	C.ACE_STATUS_GENERAL_ERROR:                      "ACE_STATUS_GENERAL_ERROR",
	C.ACE_STATUS_TIMEOUT:                            "ACE_STATUS_TIMEOUT",
	C.ACE_STATUS_OUT_OF_RESOURCES:                   "ACE_STATUS_OUT_OF_RESOURCES",
	C.ACE_STATUS_OUT_OF_MEMORY:                      "ACE_STATUS_OUT_OF_MEMORY",
	C.ACE_STATUS_OUT_OF_HANDLES:                     "ACE_STATUS_OUT_OF_HANDLES",
	C.ACE_STATUS_NOT_SUPPORTED:                      "ACE_STATUS_NOT_SUPPORTED",
	C.ACE_STATUS_NO_PERMISSION:                      "ACE_STATUS_NO_PERMISSION",
	C.ACE_STATUS_NOT_FOUND:                          "ACE_STATUS_NOT_FOUND",
	C.ACE_STATUS_NULL_POINTER:                       "ACE_STATUS_NULL_POINTER",
	C.ACE_STATUS_PARAM_OUT_OF_RANGE:                 "ACE_STATUS_PARAM_OUT_OF_RANGE",
	C.ACE_STATUS_BAD_PARAM:                          "ACE_STATUS_BAD_PARAM",
	C.ACE_STATUS_INCOMPATIBLE_PARAMS:                "ACE_STATUS_INCOMPATIBLE_PARAMS",
	C.ACE_STATUS_IO_ERROR:                           "ACE_STATUS_IO_ERROR",
	C.ACE_STATUS_TRY_AGAIN:                          "ACE_STATUS_TRY_AGAIN",
	C.ACE_STATUS_BUSY:                               "ACE_STATUS_BUSY",
	C.ACE_STATUS_DEAD_LOCK:                          "ACE_STATUS_DEAD_LOCK",
	C.ACE_STATUS_DATA_TYPE_OVERFLOW:                 "ACE_STATUS_DATA_TYPE_OVERFLOW",
	C.ACE_STATUS_BUFFER_OVERFLOW:                    "ACE_STATUS_BUFFER_OVERFLOW",
	C.ACE_STATUS_IN_PROGRESS:                        "ACE_STATUS_IN_PROGRESS",
	C.ACE_STATUS_CANCELED:                           "ACE_STATUS_CANCELED",
	C.ACE_STATUS_OWNER_DEAD:                         "ACE_STATUS_OWNER_DEAD",
	C.ACE_STATUS_UNRECOVERABLE:                      "ACE_STATUS_UNRECOVERABLE",
	C.ACE_STATUS_PORT_INVALID:                       "ACE_STATUS_PORT_INVALID",
	C.ACE_STATUS_PORT_NOT_OPEN:                      "ACE_STATUS_PORT_NOT_OPEN",
	C.ACE_STATUS_UNINITIALIZED:                      "ACE_STATUS_UNINITIALIZED",
	C.ACE_STATUS_ALREADY_INITIALIZED:                "ACE_STATUS_ALREADY_INITIALIZED",
	C.ACE_STATUS_ALREADY_EXISTS:                     "ACE_STATUS_ALREADY_EXISTS",
	C.ACE_STATUS_BELOW_THRESHOLD:                    "ACE_STATUS_BELOW_THRESHOLD",
	C.ACE_STATUS_STOPPED:                            "ACE_STATUS_STOPPED",
	C.ACE_STATUS_STORAGE_READ_FAIL:                  "ACE_STATUS_STORAGE_READ_FAIL",
	C.ACE_STATUS_STORAGE_WRITE_FAIL:                 "ACE_STATUS_STORAGE_WRITE_FAIL",
	C.ACE_STATUS_STORAGE_ERASE_FAIL:                 "ACE_STATUS_STORAGE_ERASE_FAIL",
	C.ACE_STATUS_STORAGE_FULL:                       "ACE_STATUS_STORAGE_FULL",
	C.ACE_STATUS_NOT_IMPLEMENTED:                    "ACE_STATUS_NOT_IMPLEMENTED",
	C.ACE_STATUS_RESOURCE_RECLAIMABLE:               "ACE_STATUS_RESOURCE_RECLAIMABLE",
	C.ACE_STATUS_DATA_CORRUPTED:                     "ACE_STATUS_DATA_CORRUPTED",
	C.ACE_STATUS_CONNECTED:                          "ACE_STATUS_CONNECTED",
	C.ACE_STATUS_DISCONNECTED:                       "ACE_STATUS_DISCONNECTED",
	C.ACE_STATUS_RESET:                              "ACE_STATUS_RESET",
	C.ACE_STATUS_FAILURE_UNKNOWN_FILESYSTEM:         "ACE_STATUS_FAILURE_UNKNOWN_FILESYSTEM",
	C.ACE_STATUS_FAILURE_MAX_FILESYSTEMS:            "ACE_STATUS_FAILURE_MAX_FILESYSTEMS",
	C.ACE_STATUS_FAILURE_INCOMPATIBLE_FILE:          "ACE_STATUS_FAILURE_INCOMPATIBLE_FILE",
	C.ACE_STATUS_FAILURE_FILE_NOT_OPEN:              "ACE_STATUS_FAILURE_FILE_NOT_OPEN",
	C.ACE_STATUS_EOF:                                "ACE_STATUS_EOF",
	C.ACE_STATUS_MAX_FILE_SIZE_REACHED:              "ACE_STATUS_MAX_FILE_SIZE_REACHED",
	C.ACE_STATUS_FAILURE_UNKNOWN_FILE:               "ACE_STATUS_FAILURE_UNKNOWN_FILE",
	C.ACE_STATUS_DIR_EXISTS:                         "ACE_STATUS_DIR_EXISTS",
	C.ACE_STATUS_DIR_NOT_SUPPORTED:                  "ACE_STATUS_DIR_NOT_SUPPORTED",
	C.ACE_STATUS_INVALID_PATH:                       "ACE_STATUS_INVALID_PATH",
	C.ACE_STATUS_DATA_LEN_INVALID:                   "ACE_STATUS_DATA_LEN_INVALID",
	C.ACE_STATUS_NO_NET:                             "ACE_STATUS_NO_NET",
	C.ACE_STATUS_NET_CONNECTION_ERROR:               "ACE_STATUS_NET_CONNECTION_ERROR",
	C.ACE_STATUS_NET_CONNECTION_TIMEOUT_ERROR:       "ACE_STATUS_NET_CONNECTION_TIMEOUT_ERROR",
	C.ACE_STATUS_NET_TRANSMIT_ABORT_ERROR:           "ACE_STATUS_NET_TRANSMIT_ABORT_ERROR",
	C.ACE_STATUS_NET_RECEIVE_ABORT_ERROR:            "ACE_STATUS_NET_RECEIVE_ABORT_ERROR",
	C.ACE_STATUS_NET_AUTH_FAILURE:                   "ACE_STATUS_NET_AUTH_FAILURE",
	C.ACE_STATUS_CLI_HELP_COMMAND:                   "ACE_STATUS_CLI_HELP_COMMAND",
	C.ACE_STATUS_CLI_FUNC_ERROR:                     "ACE_STATUS_CLI_FUNC_ERROR",
	C.ACE_STATUS_EVENTS_MAX_SUBSCRIBERS:             "ACE_STATUS_EVENTS_MAX_SUBSCRIBERS",
	C.ACE_STATUS_ATZ_INTERNAL_ERROR:                 "ACE_STATUS_ATZ_INTERNAL_ERROR",
	C.ACE_STATUS_DEVICE_INFO_INTERNAL_ERROR:         "ACE_STATUS_DEVICE_INFO_INTERNAL_ERROR",
	C.ACE_STATUS_DEVICE_INFO_ENTRY_NOT_SUPPORTED:    "ACE_STATUS_DEVICE_INFO_ENTRY_NOT_SUPPORTED",
	C.ACE_STATUS_MODULE_INIT_ERROR:                  "ACE_STATUS_MODULE_INIT_ERROR",
	C.ACE_STATUS_REGISTRATION_REQUEST_ERROR:         "ACE_STATUS_REGISTRATION_REQUEST_ERROR",
	C.ACE_STATUS_REGISTRATION_RESPONSE_ERROR:        "ACE_STATUS_REGISTRATION_RESPONSE_ERROR",
	C.ACE_STATUS_NOT_REGISTERED_ERROR:               "ACE_STATUS_NOT_REGISTERED_ERROR",
	C.ACE_STATUS_REGISTRATION_INVALID_INFO:          "ACE_STATUS_REGISTRATION_INVALID_INFO",
	C.ACE_STATUS_REGISTRATION_REQUEST_PAYLOAD_ERROR: "ACE_STATUS_REGISTRATION_REQUEST_PAYLOAD_ERROR",
	C.ACE_STATUS_REGISTRATION_INVALID_RESPONSE:      "ACE_STATUS_REGISTRATION_INVALID_RESPONSE",
	C.ACE_STATUS_REGISTRATION_INTERNAL_ERROR:        "ACE_STATUS_REGISTRATION_INTERNAL_ERROR",
	C.ACE_STATUS_ACCESS_TOKEN_EXPIRED:               "ACE_STATUS_ACCESS_TOKEN_EXPIRED",
	C.ACE_STATUS_INVALID_REFRESH_TOKEN:              "ACE_STATUS_INVALID_REFRESH_TOKEN",
	C.ACE_STATUS_ACM_CONNECTION_ERROR:               "ACE_STATUS_ACM_CONNECTION_ERROR",
	C.ACE_STATUS_REGISTRATION_ACCOUNT_CHALLENGED:    "ACE_STATUS_REGISTRATION_ACCOUNT_CHALLENGED",
	C.ACE_STATUS_PWR_ZERO_REF_COUNT:                 "ACE_STATUS_PWR_ZERO_REF_COUNT",
	C.ACE_STATUS_THERMAL_GETDATA_ERR:                "ACE_STATUS_THERMAL_GETDATA_ERR",
	C.ACE_STATUS_THERMAL_LOAD_POLICY_ERR:            "ACE_STATUS_THERMAL_LOAD_POLICY_ERR",
	C.ACE_STATUS_THERMAL_FUNC_ERR:                   "ACE_STATUS_THERMAL_FUNC_ERR",
	C.ACE_STATUS_PROTOCOL_ERROR:                     "ACE_STATUS_PROTOCOL_ERROR",
	C.ACE_STATUS_MORE_DATA:                          "ACE_STATUS_MORE_DATA",
	C.ACE_STATUS_BAUDRATE_INVALID:                   "ACE_STATUS_BAUDRATE_INVALID",
	C.ACE_STATUS_PARITY_INVALID:                     "ACE_STATUS_PARITY_INVALID",
	C.ACE_STATUS_STOP_BITS_INVALID:                  "ACE_STATUS_STOP_BITS_INVALID",
	C.ACE_STATUS_FLOW_CONTROL_INVALID:               "ACE_STATUS_FLOW_CONTROL_INVALID",
	C.ACE_STATUS_DEVICE_STATE_INVALID:               "ACE_STATUS_DEVICE_STATE_INVALID",
	C.ACE_STATUS_HW_FAILURE:                         "ACE_STATUS_HW_FAILURE",
	C.ACE_STATUS_DEVICE_OPERATION_ERROR:             "ACE_STATUS_DEVICE_OPERATION_ERROR",
	C.ACE_STATUS_INIT_ERROR:                         "ACE_STATUS_INIT_ERROR",
	C.ACE_STATUS_POLICY_WRITE_INVALID:               "ACE_STATUS_POLICY_WRITE_INVALID",
	C.ACE_STATUS_DEVICE_NOT_FOUND:                   "ACE_STATUS_DEVICE_NOT_FOUND",
	C.ACE_STATUS_DEVICE_NO_CONFIG:                   "ACE_STATUS_DEVICE_NO_CONFIG",
	C.ACE_STATUS_DB_OPEN_ERROR:                      "ACE_STATUS_DB_OPEN_ERROR",
	C.ACE_STATUS_BT_JNI_ENVIRONMENT_ERROR:           "ACE_STATUS_BT_JNI_ENVIRONMENT_ERROR",
	C.ACE_STATUS_BT_JNI_THREAD_ATTACH_ERROR:         "ACE_STATUS_BT_JNI_THREAD_ATTACH_ERROR",
	C.ACE_STATUS_BT_WAKELOCK_ERROR:                  "ACE_STATUS_BT_WAKELOCK_ERROR",
	C.ACE_STATUS_BT_CONN_PENDING:                    "ACE_STATUS_BT_CONN_PENDING",
	C.ACE_STATUS_BT_AUTH_FAIL_CONN_TIMEOUT:          "ACE_STATUS_BT_AUTH_FAIL_CONN_TIMEOUT",
	C.ACE_STATUS_BT_RMT_DEV_DOWN:                    "ACE_STATUS_BT_RMT_DEV_DOWN",
	C.ACE_STATUS_BT_DONE:                            "ACE_STATUS_BT_DONE",
	C.ACE_STATUS_BT_UNHANDLED:                       "ACE_STATUS_BT_UNHANDLED",
	C.ACE_STATUS_BT_AUTH_REJECTED:                   "ACE_STATUS_BT_AUTH_REJECTED",
	C.ACE_STATUS_BT_AUTH_FAIL_SMP_FAIL:              "ACE_STATUS_BT_AUTH_FAIL_SMP_FAIL",
}

var statusDescriptions = map[int32]string{
	C.ACE_STATUS_OK:                                 "Operation completed successfully",
	C.ACE_STATUS_GENERAL_ERROR:                      "Unspecified run-time error",
	C.ACE_STATUS_TIMEOUT:                            "Operation timed out",
	C.ACE_STATUS_OUT_OF_RESOURCES:                   "Resource not available",
	C.ACE_STATUS_OUT_OF_MEMORY:                      "Failed to allocate memory",
	C.ACE_STATUS_OUT_OF_HANDLES:                     "Out of file handles",
	C.ACE_STATUS_NOT_SUPPORTED:                      "Not supported on this platform",
	C.ACE_STATUS_NO_PERMISSION:                      "No permission for operation",
	C.ACE_STATUS_NOT_FOUND:                          "Indicated resource not found",
	C.ACE_STATUS_NULL_POINTER:                       "Null pointer provided",
	C.ACE_STATUS_PARAM_OUT_OF_RANGE:                 "Parameter out of range",
	C.ACE_STATUS_BAD_PARAM:                          "Parameter value bad",
	C.ACE_STATUS_INCOMPATIBLE_PARAMS:                "Parameters form incompatible set",
	C.ACE_STATUS_IO_ERROR:                           "Input/Output error",
	C.ACE_STATUS_TRY_AGAIN:                          "Safe to try again",
	C.ACE_STATUS_BUSY:                               "Resource busy",
	C.ACE_STATUS_DEAD_LOCK:                          "Mutex in dead lock",
	C.ACE_STATUS_DATA_TYPE_OVERFLOW:                 "Defined data type overflowed",
	C.ACE_STATUS_BUFFER_OVERFLOW:                    "Destination buffer overflowed",
	C.ACE_STATUS_IN_PROGRESS:                        "Operation already in progress",
	C.ACE_STATUS_CANCELED:                           "Operation canceled",
	C.ACE_STATUS_OWNER_DEAD:                         "Owner of resource died",
	C.ACE_STATUS_UNRECOVERABLE:                      "Unrecoverable error",
	C.ACE_STATUS_PORT_INVALID:                       "Invalid port",
	C.ACE_STATUS_PORT_NOT_OPEN:                      "Device port not opened",
	C.ACE_STATUS_UNINITIALIZED:                      "Resource uninitialized",
	C.ACE_STATUS_ALREADY_INITIALIZED:                "Resource already initialized",
	C.ACE_STATUS_ALREADY_EXISTS:                     "Resource already exists",
	C.ACE_STATUS_BELOW_THRESHOLD:                    "Parameter below acceptable threshold",
	C.ACE_STATUS_STOPPED:                            "Resource stopped",
	C.ACE_STATUS_STORAGE_READ_FAIL:                  "Storage read failure",
	C.ACE_STATUS_STORAGE_WRITE_FAIL:                 "Storage write failure",
	C.ACE_STATUS_STORAGE_ERASE_FAIL:                 "Storage erase failure",
	C.ACE_STATUS_STORAGE_FULL:                       "Storage is full",
	C.ACE_STATUS_NOT_IMPLEMENTED:                    "API/Operation is not implemented",
	C.ACE_STATUS_RESOURCE_RECLAIMABLE:               "Resource can be reclaimed",
	C.ACE_STATUS_DATA_CORRUPTED:                     "Data is corrupted",
	C.ACE_STATUS_CONNECTED:                          "Connected",
	C.ACE_STATUS_DISCONNECTED:                       "Disconnected",
	C.ACE_STATUS_RESET:                              "Reset occured",
	C.ACE_STATUS_FAILURE_UNKNOWN_FILESYSTEM:         "Filesystem not integrated with the system",
	C.ACE_STATUS_FAILURE_MAX_FILESYSTEMS:            "System already configured with the maximum number of filesystems",
	C.ACE_STATUS_FAILURE_INCOMPATIBLE_FILE:          "Flle operation not compatible",
	C.ACE_STATUS_FAILURE_FILE_NOT_OPEN:              "File not open",
	C.ACE_STATUS_EOF:                                "File pointer has reached end of file",
	C.ACE_STATUS_MAX_FILE_SIZE_REACHED:              "Requested size not supported",
	C.ACE_STATUS_FAILURE_UNKNOWN_FILE:               "Operation not supported",
	C.ACE_STATUS_DIR_EXISTS:                         "Directory already exists",
	C.ACE_STATUS_DIR_NOT_SUPPORTED:                  "Filesystem does not support directories",
	C.ACE_STATUS_INVALID_PATH:                       "Invalid path",
	C.ACE_STATUS_DATA_LEN_INVALID:                   "Invalid data length received",
	C.ACE_STATUS_NO_NET:                             "No network available",
	C.ACE_STATUS_NET_CONNECTION_ERROR:               "Network connection error",
	C.ACE_STATUS_NET_CONNECTION_TIMEOUT_ERROR:       "Network connection timeout error",
	C.ACE_STATUS_NET_TRANSMIT_ABORT_ERROR:           "Network transmit abort error",
	C.ACE_STATUS_NET_RECEIVE_ABORT_ERROR:            "Network transmit abort error",
	C.ACE_STATUS_NET_AUTH_FAILURE:                   "Connection failed due to authentication failure",
	C.ACE_STATUS_CLI_HELP_COMMAND:                   "Command passed to print help",
	C.ACE_STATUS_CLI_FUNC_ERROR:                     "CLI function failed",
	C.ACE_STATUS_EVENTS_MAX_SUBSCRIBERS:             "Events exceeded maximum number of subscribers",
	C.ACE_STATUS_ATZ_INTERNAL_ERROR:                 "ACE ATZ internal error",
	C.ACE_STATUS_DEVICE_INFO_INTERNAL_ERROR:         "Error internal to device information middleware module",
	C.ACE_STATUS_DEVICE_INFO_ENTRY_NOT_SUPPORTED:    "Device information entry not supported",
	C.ACE_STATUS_MODULE_INIT_ERROR:                  "Module initialization error",
	C.ACE_STATUS_REGISTRATION_REQUEST_ERROR:         "Registration request error",
	C.ACE_STATUS_REGISTRATION_RESPONSE_ERROR:        "Registration response error",
	C.ACE_STATUS_NOT_REGISTERED_ERROR:               "Device not registered error",
	C.ACE_STATUS_REGISTRATION_INVALID_INFO:          "Invalid registration info error",
	C.ACE_STATUS_REGISTRATION_REQUEST_PAYLOAD_ERROR: "Registration request payload error",
	C.ACE_STATUS_REGISTRATION_INVALID_RESPONSE:      "Registration invalid response error",
	C.ACE_STATUS_REGISTRATION_INTERNAL_ERROR:        "Registration internal error",
	C.ACE_STATUS_ACCESS_TOKEN_EXPIRED:               "Access Token Expired",
	C.ACE_STATUS_INVALID_REFRESH_TOKEN:              "Invalid refresh token",
	C.ACE_STATUS_ACM_CONNECTION_ERROR:               "ACM connection error",
	C.ACE_STATUS_REGISTRATION_ACCOUNT_CHALLENGED:    "Registration account challenged error",
	C.ACE_STATUS_PWR_ZERO_REF_COUNT:                 "Resource reference count already zero",
	C.ACE_STATUS_THERMAL_GETDATA_ERR:                "Thermal get data error",
	C.ACE_STATUS_THERMAL_LOAD_POLICY_ERR:            "Thermal load policy error",
	C.ACE_STATUS_THERMAL_FUNC_ERR:                   "Thernal cli command execution error",
	C.ACE_STATUS_PROTOCOL_ERROR:                     "Error indicating violation of the agreed exchange protocol",
	C.ACE_STATUS_MORE_DATA:                          "Indicating sender will send more data",
	C.ACE_STATUS_BAUDRATE_INVALID:                   "Invalid baud rate selected",
	C.ACE_STATUS_PARITY_INVALID:                     "Bad parity",
	C.ACE_STATUS_STOP_BITS_INVALID:                  "Device returned bad stop bits",
	C.ACE_STATUS_FLOW_CONTROL_INVALID:               "Device has bad flow control",
	C.ACE_STATUS_DEVICE_STATE_INVALID:               "Device/SW state machine invalid",
	C.ACE_STATUS_HW_FAILURE:                         "Unknown hardware failure",
	C.ACE_STATUS_DEVICE_OPERATION_ERROR:             "Device operation error",
	C.ACE_STATUS_INIT_ERROR:                         "Device initialization error",
	C.ACE_STATUS_POLICY_WRITE_INVALID:               "Power/Thermal policy write error",
	C.ACE_STATUS_DEVICE_NOT_FOUND:                   "Device not found",
	C.ACE_STATUS_DEVICE_NO_CONFIG:                   "Device not configured",
	C.ACE_STATUS_DB_OPEN_ERROR:                      "KV storage database open error",
	C.ACE_STATUS_BT_JNI_ENVIRONMENT_ERROR:           "Error status due to JNI environment",
	C.ACE_STATUS_BT_JNI_THREAD_ATTACH_ERROR:         "Error status due to JNI thread malfunction",
	C.ACE_STATUS_BT_WAKELOCK_ERROR:                  "Error status due to wakelock",
	C.ACE_STATUS_BT_CONN_PENDING:                    "Status for a pending connection",
	C.ACE_STATUS_BT_AUTH_FAIL_CONN_TIMEOUT:          "Error status due to connection timeout",
	C.ACE_STATUS_BT_RMT_DEV_DOWN:                    "Error status due remote device disconnection",
	C.ACE_STATUS_BT_DONE:                            "request already completed",
	C.ACE_STATUS_BT_UNHANDLED:                       "Error status due to unhandled operation",
	C.ACE_STATUS_BT_AUTH_REJECTED:                   "Error status due to authentication rejected by remote device",
	C.ACE_STATUS_BT_AUTH_FAIL_SMP_FAIL:              "Error status due to SMP failure",
}

type Status struct {
	int32
}

func StatusFromCode(code C.ace_status_t) Status {
	return Status{int32(code)}
}

func (s Status) Description() string {
	if desc, ok := statusDescriptions[s.int32]; ok {
		return desc
	}
	return fmt.Sprintf("Unknown status code: %d", s)
}

func (s Status) Name() string {
	if name, ok := statusNames[s.int32]; ok {
		return name
	}
	return fmt.Sprintf("Unknown status code: %d", s)
}

func (s Status) String() string {
	return fmt.Sprintf("%s{%d}", s.Name(), s.int32)
}

type AceRadioState int

const (
	RadioDisabled AceRadioState = iota
	RadioEnabled
	RadioEnabling
	RadioDisabling
)

func (a *aceAdapter) RadioState() (AceRadioState, error) {
	var radioState C.aceBT_state_t
	bleStatus := C.aceBT_getRadioState(&radioState)
	if err := errForStatus(bleStatus); err != nil {
		slog.Error("Failed to get radio state", "status", bleStatus, "error", err)
		return RadioDisabled, err
	}
	switch radioState {
	case C.ACEBT_STATE_DISABLED:
		return RadioDisabled, nil
	case C.ACEBT_STATE_ENABLED:
		return RadioEnabled, nil
	case C.ACEBT_STATE_ENABLING:
		return RadioEnabling, nil
	case C.ACEBT_STATE_DISABLING:
		return RadioDisabling, nil
	default:
		return RadioDisabled, fmt.Errorf("unknown radio state: %d", radioState)
	}
}
