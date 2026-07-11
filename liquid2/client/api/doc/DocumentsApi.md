# liquid2_api.api.DocumentsApi

## Load the API package
```dart
import 'package:liquid2_api/api.dart';
```

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**deleteDocument**](DocumentsApi.md#deletedocument) | **DELETE** /api/v1/documents/{id} | Soft-delete document
[**getDocument**](DocumentsApi.md#getdocument) | **GET** /api/v1/documents/{id} | Get document detail
[**listDocuments**](DocumentsApi.md#listdocuments) | **GET** /api/v1/documents | List documents
[**markDocumentRead**](DocumentsApi.md#markdocumentread) | **POST** /api/v1/documents/{id}/mark-read | Mark document read
[**markDocumentUnread**](DocumentsApi.md#markdocumentunread) | **POST** /api/v1/documents/{id}/mark-unread | Mark document unread
[**moveDocumentToTrash**](DocumentsApi.md#movedocumenttotrash) | **POST** /api/v1/documents/{id}/move-to-trash | Move document to trash
[**replaceDocumentTags**](DocumentsApi.md#replacedocumenttags) | **PUT** /api/v1/documents/{id}/tags | Replace document tags
[**setDocumentRating**](DocumentsApi.md#setdocumentrating) | **PUT** /api/v1/documents/{id}/rating | Set document rating
[**translateDocument**](DocumentsApi.md#translatedocument) | **POST** /api/v1/documents/{id}/translate | Translate document content
[**updateDocument**](DocumentsApi.md#updatedocument) | **PATCH** /api/v1/documents/{id} | Update document metadata


# **deleteDocument**
> DeletedOutputBody deleteDocument(id)

Soft-delete document

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentsApi();
final String id = id_example; // String | Document ID

try {
    final response = api.deleteDocument(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentsApi->deleteDocument: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |

### Return type

[**DeletedOutputBody**](DeletedOutputBody.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getDocument**
> DocumentDetail getDocument(id)

Get document detail

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentsApi();
final String id = id_example; // String | Document ID

try {
    final response = api.getDocument(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentsApi->getDocument: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |

### Return type

[**DocumentDetail**](DocumentDetail.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listDocuments**
> DocumentList listDocuments(q, status, folderId, includeFolderDescendants, tag, ratingMin, kind, sort, includeDeleted, includeTrash, limit, cursor)

List documents

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentsApi();
final String q = q_example; // String |
final String status = status_example; // String |
final String folderId = folderId_example; // String |
final bool includeFolderDescendants = true; // bool |
final String tag = tag_example; // String |
final int ratingMin = 789; // int |
final String kind = kind_example; // String |
final String sort = sort_example; // String |
final bool includeDeleted = true; // bool |
final bool includeTrash = true; // bool |
final int limit = 789; // int |
final String cursor = cursor_example; // String |

try {
    final response = api.listDocuments(q, status, folderId, includeFolderDescendants, tag, ratingMin, kind, sort, includeDeleted, includeTrash, limit, cursor);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentsApi->listDocuments: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **q** | **String**|  | [optional]
 **status** | **String**|  | [optional]
 **folderId** | **String**|  | [optional]
 **includeFolderDescendants** | **bool**|  | [optional]
 **tag** | **String**|  | [optional]
 **ratingMin** | **int**|  | [optional]
 **kind** | **String**|  | [optional]
 **sort** | **String**|  | [optional]
 **includeDeleted** | **bool**|  | [optional]
 **includeTrash** | **bool**|  | [optional]
 **limit** | **int**|  | [optional]
 **cursor** | **String**|  | [optional]

### Return type

[**DocumentList**](DocumentList.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **markDocumentRead**
> DocumentDetail markDocumentRead(id)

Mark document read

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentsApi();
final String id = id_example; // String | Document ID

try {
    final response = api.markDocumentRead(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentsApi->markDocumentRead: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |

### Return type

[**DocumentDetail**](DocumentDetail.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **markDocumentUnread**
> DocumentDetail markDocumentUnread(id)

Mark document unread

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentsApi();
final String id = id_example; // String | Document ID

try {
    final response = api.markDocumentUnread(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentsApi->markDocumentUnread: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |

### Return type

[**DocumentDetail**](DocumentDetail.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **moveDocumentToTrash**
> DocumentDetail moveDocumentToTrash(id)

Move document to trash

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentsApi();
final String id = id_example; // String | Document ID

try {
    final response = api.moveDocumentToTrash(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentsApi->moveDocumentToTrash: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |

### Return type

[**DocumentDetail**](DocumentDetail.md)

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

final api = Liquid2Api().getDocumentsApi();
final String id = id_example; // String | Document ID
final ReplaceTagsInputBody replaceTagsInputBody = ; // ReplaceTagsInputBody |

try {
    final response = api.replaceDocumentTags(id, replaceTagsInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentsApi->replaceDocumentTags: $e\n');
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

# **setDocumentRating**
> DocumentDetail setDocumentRating(id, ratingInputBody)

Set document rating

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentsApi();
final String id = id_example; // String | Document ID
final RatingInputBody ratingInputBody = ; // RatingInputBody |

try {
    final response = api.setDocumentRating(id, ratingInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentsApi->setDocumentRating: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |
 **ratingInputBody** | [**RatingInputBody**](RatingInputBody.md)|  |

### Return type

[**DocumentDetail**](DocumentDetail.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **translateDocument**
> TranslateDocumentOutputBody translateDocument(id, translateDocumentInputBody)

Translate document content

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentsApi();
final String id = id_example; // String | Document ID
final TranslateDocumentInputBody translateDocumentInputBody = ; // TranslateDocumentInputBody |

try {
    final response = api.translateDocument(id, translateDocumentInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentsApi->translateDocument: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |
 **translateDocumentInputBody** | [**TranslateDocumentInputBody**](TranslateDocumentInputBody.md)|  |

### Return type

[**TranslateDocumentOutputBody**](TranslateDocumentOutputBody.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateDocument**
> DocumentDetail updateDocument(id, updateDocumentInputBody)

Update document metadata

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentsApi();
final String id = id_example; // String | Document ID
final UpdateDocumentInputBody updateDocumentInputBody = ; // UpdateDocumentInputBody |

try {
    final response = api.updateDocument(id, updateDocumentInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentsApi->updateDocument: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |
 **updateDocumentInputBody** | [**UpdateDocumentInputBody**](UpdateDocumentInputBody.md)|  |

### Return type

[**DocumentDetail**](DocumentDetail.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
