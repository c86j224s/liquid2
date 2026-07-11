# liquid2_api.api.JobsApi

## Load the API package
```dart
import 'package:liquid2_api/api.dart';
```

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getJob**](JobsApi.md#getjob) | **GET** /api/v1/jobs/{id} | Get job
[**listJobs**](JobsApi.md#listjobs) | **GET** /api/v1/jobs | List jobs


# **getJob**
> Job getJob(id)

Get job

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getJobsApi();
final String id = id_example; // String | Job ID

try {
    final response = api.getJob(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling JobsApi->getJob: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Job ID |

### Return type

[**Job**](Job.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listJobs**
> JobList listJobs(status, kind, limit)

List jobs

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getJobsApi();
final String status = status_example; // String |
final String kind = kind_example; // String |
final int limit = 789; // int |

try {
    final response = api.listJobs(status, kind, limit);
    print(response);
} on DioException catch (e) {
    print('Exception when calling JobsApi->listJobs: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **status** | **String**|  | [optional]
 **kind** | **String**|  | [optional]
 **limit** | **int**|  | [optional]

### Return type

[**JobList**](JobList.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
