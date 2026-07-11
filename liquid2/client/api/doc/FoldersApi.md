# liquid2_api.api.FoldersApi

## Load the API package
```dart
import 'package:liquid2_api/api.dart';
```

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**createFolder**](FoldersApi.md#createfolder) | **POST** /api/v1/folders | Create folder
[**deleteFolder**](FoldersApi.md#deletefolder) | **DELETE** /api/v1/folders/{id} | Delete folder
[**listFolders**](FoldersApi.md#listfolders) | **GET** /api/v1/folders | List folder tree
[**updateFolder**](FoldersApi.md#updatefolder) | **PATCH** /api/v1/folders/{id} | Update folder


# **createFolder**
> Folder createFolder(folderBodyInputBody)

Create folder

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getFoldersApi();
final FolderBodyInputBody folderBodyInputBody = ; // FolderBodyInputBody |

try {
    final response = api.createFolder(folderBodyInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling FoldersApi->createFolder: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **folderBodyInputBody** | [**FolderBodyInputBody**](FolderBodyInputBody.md)|  |

### Return type

[**Folder**](Folder.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deleteFolder**
> deleteFolder(id, documentAction)

Delete folder

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getFoldersApi();
final String id = id_example; // String | Folder ID
final String documentAction = documentAction_example; // String |

try {
    api.deleteFolder(id, documentAction);
} on DioException catch (e) {
    print('Exception when calling FoldersApi->deleteFolder: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Folder ID |
 **documentAction** | **String**|  | [optional]

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listFolders**
> FolderListOutputBody listFolders()

List folder tree

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getFoldersApi();

try {
    final response = api.listFolders();
    print(response);
} on DioException catch (e) {
    print('Exception when calling FoldersApi->listFolders: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**FolderListOutputBody**](FolderListOutputBody.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateFolder**
> Folder updateFolder(id, updateFolderInputBody)

Update folder

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getFoldersApi();
final String id = id_example; // String | Folder ID
final UpdateFolderInputBody updateFolderInputBody = ; // UpdateFolderInputBody |

try {
    final response = api.updateFolder(id, updateFolderInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling FoldersApi->updateFolder: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Folder ID |
 **updateFolderInputBody** | [**UpdateFolderInputBody**](UpdateFolderInputBody.md)|  |

### Return type

[**Folder**](Folder.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
