# liquid2_api.api.ExportApi

## Load the API package
```dart
import 'package:liquid2_api/api.dart';
```

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**createExport**](ExportApi.md#createexport) | **POST** /api/v1/export | Create markdown export
[**getExport**](ExportApi.md#getexport) | **GET** /api/v1/exports/{id} | Get export metadata


# **createExport**
> ExportOutputBody createExport(createExportInputBody)

Create markdown export

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getExportApi();
final CreateExportInputBody createExportInputBody = ; // CreateExportInputBody |

try {
    final response = api.createExport(createExportInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling ExportApi->createExport: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **createExportInputBody** | [**CreateExportInputBody**](CreateExportInputBody.md)|  |

### Return type

[**ExportOutputBody**](ExportOutputBody.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getExport**
> ExportOutputBody getExport(id)

Get export metadata

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getExportApi();
final String id = id_example; // String | Export artifact ID

try {
    final response = api.getExport(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling ExportApi->getExport: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Export artifact ID |

### Return type

[**ExportOutputBody**](ExportOutputBody.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
