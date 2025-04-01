<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->
# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [status/proto/agent_status.proto](#status_proto_agent_status-proto)
    - [GetStatusIntervalRequest](#agent_status_proto-v1-GetStatusIntervalRequest)
    - [GetStatusIntervalResponse](#agent_status_proto-v1-GetStatusIntervalResponse)
    - [ReportStatusRequest](#agent_status_proto-v1-ReportStatusRequest)
    - [ReportStatusResponse](#agent_status_proto-v1-ReportStatusResponse)
  
    - [Status](#agent_status_proto-v1-Status)
  
    - [StatusService](#agent_status_proto-v1-StatusService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="status_proto_agent_status-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## status/proto/agent_status.proto



<a name="agent_status_proto-v1-GetStatusIntervalRequest"></a>

### GetStatusIntervalRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| agent_name | [string](#string) |  | Agent name for future use for registration |






<a name="agent_status_proto-v1-GetStatusIntervalResponse"></a>

### GetStatusIntervalResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| interval_seconds | [int32](#int32) |  | Interval in seconds |






<a name="agent_status_proto-v1-ReportStatusRequest"></a>

### ReportStatusRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| agent_name | [string](#string) |  | Agent name to optionally call out on UI |
| status | [Status](#agent_status_proto-v1-Status) |  | Binary ready/non-ready |
| detail | [google.protobuf.Struct](#google-protobuf-Struct) |  | For future use if the status is complex |






<a name="agent_status_proto-v1-ReportStatusResponse"></a>

### ReportStatusResponse






 


<a name="agent_status_proto-v1-Status"></a>

### Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| STATUS_READY | 1 |  |
| STATUS_NOT_READY | 2 |  |


 

 


<a name="agent_status_proto-v1-StatusService"></a>

### StatusService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ReportStatus | [ReportStatusRequest](#agent_status_proto-v1-ReportStatusRequest) | [ReportStatusResponse](#agent_status_proto-v1-ReportStatusResponse) |  |
| GetStatusInterval | [GetStatusIntervalRequest](#agent_status_proto-v1-GetStatusIntervalRequest) | [GetStatusIntervalResponse](#agent_status_proto-v1-GetStatusIntervalResponse) |  |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

