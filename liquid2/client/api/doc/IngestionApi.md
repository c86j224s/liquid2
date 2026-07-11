# liquid2_api.api.IngestionApi

## Load the API package
```dart
import 'package:liquid2_api/api.dart';
```

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**bookmarkDocument**](IngestionApi.md#bookmarkdocument) | **POST** /api/v1/documents/bookmark | Bookmark URL
[**rescrapeDocument**](IngestionApi.md#rescrapedocument) | **POST** /api/v1/documents/{id}/rescrape | Re-scrape document
[**scrapeDocument**](IngestionApi.md#scrapedocument) | **POST** /api/v1/documents/scrape | Scrape URL
[**scrapeTranslateDocument**](IngestionApi.md#scrapetranslatedocument) | **POST** /api/v1/documents/scrape-translate | Scrape URL and translate
[**uploadDocument**](IngestionApi.md#uploaddocument) | **POST** /api/v1/documents/upload | Upload document


# **bookmarkDocument**
> DocumentDetail bookmarkDocument(bookmarkDocumentInputBody)

Bookmark URL

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getIngestionApi();
final BookmarkDocumentInputBody bookmarkDocumentInputBody = ; // BookmarkDocumentInputBody |

try {
    final response = api.bookmarkDocument(bookmarkDocumentInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling IngestionApi->bookmarkDocument: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **bookmarkDocumentInputBody** | [**BookmarkDocumentInputBody**](BookmarkDocumentInputBody.md)|  |

### Return type

[**DocumentDetail**](DocumentDetail.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **rescrapeDocument**
> DocumentDetail rescrapeDocument(id)

Re-scrape document

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getIngestionApi();
final String id = id_example; // String | Document ID

try {
    final response = api.rescrapeDocument(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling IngestionApi->rescrapeDocument: $e\n');
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

# **scrapeDocument**
> DocumentDetail scrapeDocument(scrapeDocumentInputBody)

Scrape URL

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getIngestionApi();
final ScrapeDocumentInputBody scrapeDocumentInputBody = ; // ScrapeDocumentInputBody |

try {
    final response = api.scrapeDocument(scrapeDocumentInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling IngestionApi->scrapeDocument: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **scrapeDocumentInputBody** | [**ScrapeDocumentInputBody**](ScrapeDocumentInputBody.md)|  |

### Return type

[**DocumentDetail**](DocumentDetail.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **scrapeTranslateDocument**
> ScrapeTranslateDocumentOutputBody scrapeTranslateDocument(scrapeTranslateDocumentInputBody)

Scrape URL and translate

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getIngestionApi();
final ScrapeTranslateDocumentInputBody scrapeTranslateDocumentInputBody = ; // ScrapeTranslateDocumentInputBody |

try {
    final response = api.scrapeTranslateDocument(scrapeTranslateDocumentInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling IngestionApi->scrapeTranslateDocument: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **scrapeTranslateDocumentInputBody** | [**ScrapeTranslateDocumentInputBody**](ScrapeTranslateDocumentInputBody.md)|  |

### Return type

[**ScrapeTranslateDocumentOutputBody**](ScrapeTranslateDocumentOutputBody.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **uploadDocument**
> DocumentDetail uploadDocument(file, folderId, tagIds, title)

Upload document

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getIngestionApi();
final MultipartFile file = BINARY_DATA_HERE; // MultipartFile |
final String folderId = folderId_example; // String |
final BuiltList<String> tagIds = ; // BuiltList<String> |
final String title = title_example; // String |

try {
    final response = api.uploadDocument(file, folderId, tagIds, title);
    print(response);
} on DioException catch (e) {
    print('Exception when calling IngestionApi->uploadDocument: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **file** | **MultipartFile**|  |
 **folderId** | **String**|  | [optional]
 **tagIds** | [**BuiltList&lt;String&gt;**](String.md)|  | [optional]
 **title** | **String**|  | [optional]

### Return type

[**DocumentDetail**](DocumentDetail.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: multipart/form-data
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
