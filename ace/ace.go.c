#include "ace.go.h"

aceBT_sessionCallbacks_t session_callbacks = {
    .size = sizeof(aceBT_sessionCallbacks_t),
    .session_state_cb = onSessionStateChanged,
};

aceBT_beaconCallbacks_t beacon_callbacks = {
    .size = sizeof(aceBT_beaconCallbacks_t),
    // Advertisement state changed
    .advStateChanged = advChangeCallback,
    // Scan state changed
    .scanStateChanged = scanChangeCallback,
    // Scan results callback
    .scanResults = scanResultCallback,
    // Beacon client registration callback
    .onclientRegistered = onBeaconClientRegistered,
};

aceBT_bleCallbacks_t ble_callbacks = {
    .size = sizeof(aceBT_bleCallbacks_t),
    .common_cbs = {
        .size = sizeof(aceBT_commonCallbacks_t),
        .adapter_state_cb = onAdapterStateChanged,
        .bond_state_cb = onBondStateChanged,
        .acl_state_changed_cb = NULL,
    },
    .ble_registered_cb = onBleRegistered,
    .connection_state_change_cb = onBleConnectionStateChanged,
};

aceBT_bleGattClientCallbacks_t ble_gatt_client_callbacks = {
    .size = sizeof(aceBT_bleGattClientCallbacks_t),
    // Service registered callback
    .on_ble_gattc_service_registered_cb = onBleGattcServiceRegistered,
    // Service discovered callback
    .on_ble_gattc_service_discovered_cb = onBleGattcServiceDiscovered,
    // Read characteristics callback
    .on_ble_gattc_read_characteristics_cb = onBleGattcReadCharacteristics,
    // Write characteristics callback
    .on_ble_gattc_write_characteristics_cb = onBleGattcWriteCharacteristics,
    /// Characteristics notification callback
    .notify_characteristics_cb = onBleGattcNotifyCharacteristics,
    // Write characteristics callback
    .on_ble_gattc_write_descriptor_cb = onBleGattcWriteDescriptor,
    // Read characteristics callback for GATTC descriptor
    .on_ble_gattc_read_descriptor_cb = onBleGattcReadDescriptor,
    // Get GATT database callback
    .on_ble_gattc_get_gatt_db_cb = onBleGattcGetGattDb,
    // Execute write callback
    .on_ble_gattc_execute_write_cb = onBleGattcExecuteWrite,
};

// Exists to work around cgo deficiencies with struct layout: aceBT_deviceList_t
// doesn't correctly expose p_devices with cgo and packed structs.
cgo_deviceList cgo_getDeviceList(aceBT_deviceList_t *device_list)
{
    if (device_list == NULL || device_list->num_devices == 0)
    {
        return (cgo_deviceList){};
    }
    cgo_deviceList list = {
        .num_devices = device_list->num_devices,
        .p_devices = device_list->p_devices,
    };
    return list;
}

void cgo_getUUIDFromGATTCharRecord(aceBT_bleGattCharacteristicsValue_t *char_val, uint8_t uuid[16])
{
    if (char_val == NULL || uuid == NULL)
    {
        return;
    }
    // Copy the UUID from the GATT characteristic record to the output buffer
    memcpy(uuid, char_val->gattRecord.uuid.uu, 16);
}

void cgo_getRecordFromChar(aceBT_bleGattCharacteristicsValue_t *char_val, aceBT_bleGattRecord_t *record)
{
    if (char_val == NULL || record == NULL)
    {
        return;
    }
    memcpy(record, &char_val->gattRecord, sizeof(aceBT_bleGattRecord_t));
}

void cgo_getDescriptorFromChar(aceBT_bleGattCharacteristicsValue_t *char_val, aceBT_bleGattDescriptor_t *desc)
{
    if (char_val == NULL || desc == NULL)
    {
        return;
    }
    memcpy(desc, &char_val->gattDescriptor, sizeof(aceBT_bleGattDescriptor_t));
    // fprintf(stderr, "XXX: is_notify=%d is_set=%d write_type=%d\n", desc->is_notify, desc->is_set, desc->write_type);
}

ace_status_t cgo_bleWriteCharacteristics(aceBT_sessionHandle session_handle, aceBT_bleConnHandle conn_handle,
    aceBT_bleGattCharacteristicsValue_t *chars_value, aceBT_responseType_t request_type,
    uint8_t *data, size_t data_len)
{
    if (session_handle == NULL || conn_handle == NULL || chars_value == NULL || data == NULL)
    {
        if (session_handle == NULL)
        {
            fprintf(stderr, "cgo_bleWriteCharacteristics: session_handle is NULL\n");
        }
        if (conn_handle == NULL)
        {
            fprintf(stderr, "cgo_bleWriteCharacteristics: conn_handle is NULL\n");
        }
        if (chars_value == NULL)
        {
            fprintf(stderr, "cgo_bleWriteCharacteristics: chars_value is NULL\n");
        }
        if (data == NULL)
        {
            fprintf(stderr, "cgo_bleWriteCharacteristics: data is NULL\n");
        }
        return ACE_STATUS_BAD_PARAM;
    }
    // TODO: do we ever write small enough for the optimized path?
    chars_value->format = ACEBT_BLE_FORMAT_BLOB;
    chars_value->blobValue.offset = 0;
    chars_value->blobValue.size = data_len;
    chars_value->blobValue.data = malloc(data_len);
    if (chars_value->blobValue.data == NULL)
    {
        fprintf(stderr, "cgo_bleWriteCharacteristics: malloc failed, %p\n", chars_value->blobValue.data);
        return ACE_STATUS_OUT_OF_MEMORY;
    }
    memcpy(chars_value->blobValue.data, data, data_len);
    // fprintf(stderr, "XXXXX: session_handle=%p conn_handle=%p chars_value=%p request_type=%d data=%p data_len=%zu\n",
    //     session_handle, conn_handle, chars_value,
    //     request_type, data, data_len);
    // fprintf(stderr, "XXXXX: data = ");
    for (int i = 0; i < data_len && i < 20; i++)
    {
        fprintf(stderr, "%02x ", chars_value->blobValue.data[i]);
    }
    fprintf(stderr, "\n");
    fprintf(stderr, "blob avalue size = %d\n", chars_value->blobValue.size);
    ace_status_t status = aceBT_bleWriteCharacteristics(session_handle, conn_handle, chars_value, request_type);
    free(chars_value->blobValue.data);
    return status;
}

ace_status_t cgo_bleSetNotification(
    aceBT_sessionHandle session_handle,
    aceBT_bleConnHandle conn_handle,
    aceBT_bleGattCharacteristicsValue_t *chars_value,
    bool is_enabled
) {
    return aceBT_bleSetNotification(
        session_handle, conn_handle, *chars_value, is_enabled
    );
}


void printUuid(const aceBT_uuid_t *uuid)
{
    fprintf(stderr, "%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%"
                    "02x%02x%02x",
            uuid->uu[15], uuid->uu[14], uuid->uu[13], uuid->uu[12], uuid->uu[11],
            uuid->uu[10], uuid->uu[9], uuid->uu[8], uuid->uu[7], uuid->uu[6],
            uuid->uu[5], uuid->uu[4], uuid->uu[3], uuid->uu[2], uuid->uu[1],
            uuid->uu[0]);
}

cgo_charsValueData getDataFromCharsValue(aceBT_bleGattCharacteristicsValue_t *value) {
    cgo_charsValueData data = {0};
    if (value == NULL || value->format != ACEBT_BLE_FORMAT_BLOB) {
        return data;
    }
    data.len = value->blobValue.size;
    data.data = value->blobValue.data;
    return data;
}

void dumpCharValue(aceBT_bleGattCharacteristicsValue_t *value) {
    uint8_t format = value->format;
    if (format == ACEBT_BLE_FORMAT_UINT8) {
        fprintf(stderr, "UINT8: %d\n", value->uint8Val);
    } else if (format == ACEBT_BLE_FORMAT_UINT16) {
        fprintf(stderr, "UINT16: %d\n", value->uint16Val);
    } else if (format == ACEBT_BLE_FORMAT_UINT32) {
        fprintf(stderr, "UINT32: %d\n", value->uint32Val);
    } else if (format == ACEBT_BLE_FORMAT_SINT8) {
        fprintf(stderr, "SINT8: %d\n", value->int8Val);
    } else if (format == ACEBT_BLE_FORMAT_SINT16) {
        fprintf(stderr, "SINT16: %d\n", value->int16Val);
    } else if (format == ACEBT_BLE_FORMAT_SINT32) {
        fprintf(stderr, "SINT32: %d\n", value->int32Val);
    } else if (format == ACEBT_BLE_FORMAT_SFLOAT) {
        fprintf(stderr, "SFLOAT: %d\n", value->uint16Val);
    } else if (format == ACEBT_BLE_FORMAT_FLOAT) {
        fprintf(stderr, "FLOAT: %d\n", value->uint32Val);
    } else if (format == ACEBT_BLE_FORMAT_BLOB) {
        fprintf(stderr, "BLOB: size=%d offset=%d data=", value->blobValue.size, value->blobValue.offset);
        for (int i = 0; i < value->blobValue.size && i < 20; i++) {
            fprintf(stderr, "%02x ", value->blobValue.data[i]);
        }
        fprintf(stderr, "\n");
    } else {
        fprintf(stderr, "Unknown format: 0x%02x\n", format);
    }
}


void cgo_dumpChar(aceBT_bleGattsService_t *service)
{
    if (service == NULL)
    {
        return;
    }

    int char_count = 0;
    for (struct aceBT_gattCharRec_t *char_rec = service->charsList.stqh_first;
         char_rec != NULL;
         char_rec = char_rec->link.stqe_next)
    {

        if (char_rec->value.gattDescriptor.is_notify && char_rec->value.gattDescriptor.is_set)
        {
            fprintf(stderr, "\tGatt Characteristics with Notifications %d uuid ",
                    char_count++);
            printUuid(&char_rec->value.gattRecord.uuid);
            fprintf(stderr, "\n");
        }
        else
        {
            fprintf(stderr, "\tGatt Characteristics %d uuid ", char_count++);
            printUuid(&char_rec->value.gattRecord.uuid);
            fprintf(stderr, "\n");
        }

        if (char_rec->value.gattDescriptor.is_set)
        {
            fprintf(stderr, "\t\tDescriptor UUID ");
            printUuid(&char_rec->value.gattDescriptor.gattRecord.uuid);
            fprintf(stderr, "\n");
        }
        else if (char_rec->value.multiDescCount)
        {
            uint8_t desc_num = 1;
            struct aceBT_gattDescRec_t *desc_rec = NULL;
            for (desc_rec = char_rec->value.descList.stqh_first;
                 desc_rec != NULL;
                 desc_rec = desc_rec->link.stqe_next)
            {
                fprintf(stderr, "\t\tDescriptor %d UUID %s", desc_num++);
                printUuid(&desc_rec->value.gattRecord.uuid);
                fprintf(stderr, "\n");
            }
        }
    }
}

void cgo_dumpChars(aceBT_bleGattsService_t *service)
{
    if (service == NULL)
    {
        return;
    }

    int char_count = 0;
    for (struct aceBT_gattCharRec_t *char_rec = service->charsList.stqh_first;
         char_rec != NULL;
         char_rec = char_rec->link.stqe_next)
    {

        if (char_rec->value.gattDescriptor.is_notify && char_rec->value.gattDescriptor.is_set)
        {
            fprintf(stderr, "\tGatt Characteristics with Notifications %d uuid ",
                    char_count++);
            printUuid(&char_rec->value.gattRecord.uuid);
            fprintf(stderr, "\n");
        }
        else
        {
            fprintf(stderr, "\tGatt Characteristics %d uuid ", char_count++);
            printUuid(&char_rec->value.gattRecord.uuid);
            fprintf(stderr, "\n");
        }

        if (char_rec->value.gattDescriptor.is_set)
        {
            fprintf(stderr, "\t\tDescriptor UUID ");
            printUuid(&char_rec->value.gattDescriptor.gattRecord.uuid);
            fprintf(stderr, "\n");
        }
        else if (char_rec->value.multiDescCount)
        {
            uint8_t desc_num = 1;
            struct aceBT_gattDescRec_t *desc_rec = NULL;
            for (desc_rec = char_rec->value.descList.stqh_first;
                 desc_rec != NULL;
                 desc_rec = desc_rec->link.stqe_next)
            {
                fprintf(stderr, "\t\tDescriptor %d UUID %s", desc_num++);
                printUuid(&desc_rec->value.gattRecord.uuid);
                fprintf(stderr, "\n");
            }
        }
    }
}
