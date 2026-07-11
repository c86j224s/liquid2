# liquid2_api.api.SettingsApi

## Load the API package
```dart
import 'package:liquid2_api/api.dart';
```

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getSettings**](SettingsApi.md#getsettings) | **GET** /api/v1/settings | Get app settings
[**updateSettings**](SettingsApi.md#updatesettings) | **PATCH** /api/v1/settings | Update app settings


# **getSettings**
> AppSettings getSettings()

Get app settings

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getSettingsApi();

try {
    final response = api.getSettings();
    print(response);
} on DioException catch (e) {
    print('Exception when calling SettingsApi->getSettings: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**AppSettings**](AppSettings.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateSettings**
> AppSettings updateSettings(updateSettingsInput)

Update app settings

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getSettingsApi();
final UpdateSettingsInput updateSettingsInput = ; // UpdateSettingsInput |

try {
    final response = api.updateSettings(updateSettingsInput);
    print(response);
} on DioException catch (e) {
    print('Exception when calling SettingsApi->updateSettings: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **updateSettingsInput** | [**UpdateSettingsInput**](UpdateSettingsInput.md)|  |

### Return type

[**AppSettings**](AppSettings.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
