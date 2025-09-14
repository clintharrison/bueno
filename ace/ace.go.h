#pragma once

#include "ace/ace_status.h"
#include "ace/bluetooth_ble_api.h"
#include "ace/bluetooth_ble_defines.h"
#include "ace/bluetooth_session_api.h"
#include "ace/bluetooth_common_api.h"
#include "ace/bluetooth_beacon_api.h"
#include "ace/bluetooth_ble_gatt_client_api.h"

// this is defined in ace.h, but that brings in os_specific.h which we don't
// have a replacement for
extern ace_status_t ace_init();

/**
 * @brief callback to notifiy a change in advertisment instance\n
 * Invoked on @ref aceBT_startBeacon, @ref aceBT_startBeaconWithScanResponse,
 * and @ref aceBT_stopBeacon
 *
 * @param[in] adv_instance Advertisement instance
 * @param[in] state Current advertisement state
 * @param[in] power_mode Current power mode used for this advertisement
 * @param[in] beacon_mode Beacon mode in which this adv instance is being broadcasted
 *     typedef void (*beacon_advChangeCallback)(aceBT_advInstanceHandle adv_instance,
 *                                              aceBT_beaconAdvState_t state,
 *                                              aceBT_beaconPowerMode_t power_mode,
 *                                              aceBT_beaconAdvMode_t beacon_mode);
 */
extern void advChangeCallback(aceBT_advInstanceHandle adv_instance, aceBT_beaconAdvState_t state, aceBT_beaconPowerMode_t power_mode, aceBT_beaconAdvMode_t beacon_mode);

/**
 * @brief callback to notifiy a change in advertisment instance\n
 * Invoked on @ref aceBT_startBeaconScan, @ref
 * aceBT_startBeaconScanWithDefaultParams, @ref aceBT_stopBeaconScan
 *
 * @param[in] scan_instance Scan instance
 * @param[in] state Current advertisement state
 * @param[in] interval Interval in in untis of 1.25 ms at which this scan is
 * performed currently
 * @param[in] window length of scan procedure / scan interval in untis of 1.25
 * ms
 *     typedef void (*beacon_scanChangeCallback)(
 *         aceBT_scanInstanceHandle scan_instance, aceBT_beaconScanState_t state,
 *         uint32_t interval, uint32_t window);
 */
extern void scanChangeCallback(aceBT_scanInstanceHandle scan_instance, aceBT_beaconScanState_t state, uint32_t interval, uint32_t window);

/**
 * @brief callback to notifiy a change in advertisment instance\n
 * Invoked in response of @ref aceBT_startBeaconScan and @ref
 * aceBT_startBeaconScanWithDefaultParams
 *
 * @param[in] scan_instance Scan instance
 * @param[in] state Current advertisement state
 * @param[in] scanResult Scan result(s)
 */
extern void scanResultCallback(aceBT_scanInstanceHandle scan_instance, aceBT_BeaconScanRecord_t* record);

/**
 * @brief callback to notifiy that beacon client registration status\n
 * Invoked on @ref aceBT_RegisterBeaconClient
 *
 * @param[in] status status of the beacon client registration
 *     typedef void (*beacon_onBeaconClientRegistered)(aceBT_status_t status);
 */
extern void onBeaconClientRegistered(aceBT_status_t status);

extern void onSessionStateChanged(aceBT_sessionHandle session_handle, aceBT_sessionState_t state);

extern void onBleConnectionStateChanged(
    aceBT_bleConnState_t state, aceBT_gattStatus_t status,
    const aceBT_bleConnHandle conn_handle, aceBT_bdAddr_t* p_addr);

extern void onBleGattcServiceDiscovered(aceBT_bleConnHandle conn_handle, aceBT_status_t status);
extern void onAdapterStateChanged(aceBT_state_t state);
extern void onBondStateChanged(aceBT_status_t status, aceBT_bdAddr_t* p_remote_addr, aceBT_bondState_t state);
extern void onBleRegistered(aceBT_status_t status);

// GATT client callbacks
extern void onBleGattcServiceRegistered(aceBT_status_t status);
extern void onBleGattcReadCharacteristics(
    aceBT_bleConnHandle conn_handle,
    aceBT_bleGattCharacteristicsValue_t chars_value, aceBT_status_t status);
extern void onBleGattcWriteCharacteristics(
    aceBT_bleConnHandle conn_handle,
    aceBT_bleGattCharacteristicsValue_t gatt_characteristics,
    aceBT_status_t status);
extern void onBleGattcNotifyCharacteristics(
    aceBT_bleConnHandle conn_handle,
    aceBT_bleGattCharacteristicsValue_t gatt_characteristics);
extern void onBleGattcWriteDescriptor(
    aceBT_bleConnHandle conn_handle,
    aceBT_bleGattCharacteristicsValue_t gatt_characteristics,
    aceBT_status_t status);
extern void onBleGattcReadDescriptor(
    aceBT_bleConnHandle conn_handle,
    aceBT_bleGattCharacteristicsValue_t chars_value, aceBT_status_t status);
extern void onBleGattcGetGattDb(
    aceBT_bleConnHandle conn_handle, aceBT_bleGattsService_t* gatt_service,
    uint32_t no_svc);
extern void onBleGattcExecuteWrite(
    aceBT_bleConnHandle conn_handle, aceBT_status_t status);

extern aceBT_sessionCallbacks_t session_callbacks;
extern aceBT_bleCallbacks_t ble_callbacks;
extern aceBT_beaconCallbacks_t beacon_callbacks;
extern aceBT_bleGattClientCallbacks_t ble_gatt_client_callbacks;

// This is working around cgo limitations with C structs :(
typedef struct {
    uint16_t num_devices;
    aceBT_bdAddr_t *p_devices;
} cgo_deviceList;
extern cgo_deviceList cgo_getDeviceList(aceBT_deviceList_t *device_list);
extern void cgo_getUUIDFromGATTCharRecord(aceBT_bleGattCharacteristicsValue_t *char_val, uint8_t uuid[16]);
extern void cgo_getRecordFromChar(aceBT_bleGattCharacteristicsValue_t *char_val, aceBT_bleGattRecord_t *record);
extern void cgo_getDescriptorFromChar(aceBT_bleGattCharacteristicsValue_t *char_val, aceBT_bleGattDescriptor_t *desc);
extern ace_status_t cgo_bleWriteCharacteristics(aceBT_sessionHandle session_handle, aceBT_bleConnHandle conn_handle,
    aceBT_bleGattCharacteristicsValue_t *chars_value, aceBT_responseType_t request_type,
    uint8_t *data, size_t data_len);
extern ace_status_t cgo_bleSetNotification(
    aceBT_sessionHandle session_handle,
    aceBT_bleConnHandle conn_handle,
    aceBT_bleGattCharacteristicsValue_t *chars_value,
    bool is_enabled
);

typedef struct {
    uint8_t *data;
    size_t len;
} cgo_charsValueData;
extern cgo_charsValueData getDataFromCharsValue(aceBT_bleGattCharacteristicsValue_t *value);

// Debugging
extern void cgo_dumpChars(aceBT_bleGattsService_t *service);
extern void dumpCharValue(aceBT_bleGattCharacteristicsValue_t *value);
