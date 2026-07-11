# liquid2_api.api.TagsApi

## Load the API package
```dart
import 'package:liquid2_api/api.dart';
```

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**createTag**](TagsApi.md#createtag) | **POST** /api/v1/tags | Create tag
[**listTags**](TagsApi.md#listtags) | **GET** /api/v1/tags | List tags
[**replaceDocumentTags**](TagsApi.md#replacedocumenttags) | **PUT** /api/v1/documents/{id}/tags | Replace document tags


# **createTag**
> Tag createTag(tagBodyInputBody)

Create tag

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getTagsApi();
final TagBodyInputBody tagBodyInputBody = ; // TagBodyInputBody |

try {
    final response = api.createTag(tagBodyInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling TagsApi->createTag: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tagBodyInputBody** | [**TagBodyInputBody**](TagBodyInputBody.md)|  |

### Return type

[**Tag**](Tag.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listTags**
> TagListOutputBody listTags()

List tags

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getTagsApi();

try {
    final response = api.listTags();
    print(response);
} on DioException catch (e) {
    print('Exception when calling TagsApi->listTags: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**TagListOutputBody**](TagListOutputBody.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **replaceDocumentTags**
> DocumentDetail replaceDocumentTags(id, replaceTagsInputBody)

Replace document tags

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getTagsApi();
final String id = id_example; // String | Document ID
final ReplaceTagsInputBody replaceTagsInputBody = ; // ReplaceTagsInputBody |

try {
    final response = api.replaceDocumentTags(id, replaceTagsInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling TagsApi->replaceDocumentTags: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |
 **replaceTagsInputBody** | [**ReplaceTagsInputBody**](ReplaceTagsInputBody.md)|  |

### Return type

[**DocumentDetail**](DocumentDetail.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
