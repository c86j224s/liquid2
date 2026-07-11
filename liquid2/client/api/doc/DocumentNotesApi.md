# liquid2_api.api.DocumentNotesApi

## Load the API package
```dart
import 'package:liquid2_api/api.dart';
```

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**createDocumentNote**](DocumentNotesApi.md#createdocumentnote) | **POST** /api/v1/documents/{id}/notes | Create document note
[**deleteDocumentNote**](DocumentNotesApi.md#deletedocumentnote) | **DELETE** /api/v1/documents/{id}/notes/{noteId} | Soft-delete document note
[**listDocumentNotes**](DocumentNotesApi.md#listdocumentnotes) | **GET** /api/v1/documents/{id}/notes | List document notes
[**updateDocumentNote**](DocumentNotesApi.md#updatedocumentnote) | **PATCH** /api/v1/documents/{id}/notes/{noteId} | Update document note


# **createDocumentNote**
> DocumentNote createDocumentNote(id, noteBodyInputBody)

Create document note

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentNotesApi();
final String id = id_example; // String | Document ID
final NoteBodyInputBody noteBodyInputBody = ; // NoteBodyInputBody |

try {
    final response = api.createDocumentNote(id, noteBodyInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentNotesApi->createDocumentNote: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |
 **noteBodyInputBody** | [**NoteBodyInputBody**](NoteBodyInputBody.md)|  |

### Return type

[**DocumentNote**](DocumentNote.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deleteDocumentNote**
> DeletedOutputBody deleteDocumentNote(id, noteId)

Soft-delete document note

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentNotesApi();
final String id = id_example; // String | Document ID
final String noteId = noteId_example; // String | Note ID

try {
    final response = api.deleteDocumentNote(id, noteId);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentNotesApi->deleteDocumentNote: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |
 **noteId** | **String**| Note ID |

### Return type

[**DeletedOutputBody**](DeletedOutputBody.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listDocumentNotes**
> NoteList listDocumentNotes(id)

List document notes

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentNotesApi();
final String id = id_example; // String | Document ID

try {
    final response = api.listDocumentNotes(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentNotesApi->listDocumentNotes: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |

### Return type

[**NoteList**](NoteList.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateDocumentNote**
> DocumentNote updateDocumentNote(id, noteId, updateNoteInputBody)

Update document note

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getDocumentNotesApi();
final String id = id_example; // String | Document ID
final String noteId = noteId_example; // String | Note ID
final UpdateNoteInputBody updateNoteInputBody = ; // UpdateNoteInputBody |

try {
    final response = api.updateDocumentNote(id, noteId, updateNoteInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling DocumentNotesApi->updateDocumentNote: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Document ID |
 **noteId** | **String**| Note ID |
 **updateNoteInputBody** | [**UpdateNoteInputBody**](UpdateNoteInputBody.md)|  |

### Return type

[**DocumentNote**](DocumentNote.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
