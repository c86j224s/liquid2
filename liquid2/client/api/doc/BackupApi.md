# liquid2_api.api.BackupApi

## Load the API package
```dart
import 'package:liquid2_api/api.dart';
```

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**createBackup**](BackupApi.md#createbackup) | **POST** /api/v1/backup | Create SQLite backup


# **createBackup**
> BackupOutputBody createBackup(body)

Create SQLite backup

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getBackupApi();
final JsonObject body = Object; // JsonObject |

try {
    final response = api.createBackup(body);
    print(response);
} on DioException catch (e) {
    print('Exception when calling BackupApi->createBackup: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | **JsonObject**|  |

### Return type

[**BackupOutputBody**](BackupOutputBody.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
