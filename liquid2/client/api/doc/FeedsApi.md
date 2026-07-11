# liquid2_api.api.FeedsApi

## Load the API package
```dart
import 'package:liquid2_api/api.dart';
```

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**createFeed**](FeedsApi.md#createfeed) | **POST** /api/v1/feeds | Create feed
[**deleteFeed**](FeedsApi.md#deletefeed) | **DELETE** /api/v1/feeds/{id} | Delete feed
[**listFeeds**](FeedsApi.md#listfeeds) | **GET** /api/v1/feeds | List feeds
[**refreshFeed**](FeedsApi.md#refreshfeed) | **POST** /api/v1/feeds/{id}/refresh | Refresh feed
[**updateFeed**](FeedsApi.md#updatefeed) | **PATCH** /api/v1/feeds/{id} | Update feed


# **createFeed**
> Feed createFeed(createFeedInputBody)

Create feed

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getFeedsApi();
final CreateFeedInputBody createFeedInputBody = ; // CreateFeedInputBody |

try {
    final response = api.createFeed(createFeedInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling FeedsApi->createFeed: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **createFeedInputBody** | [**CreateFeedInputBody**](CreateFeedInputBody.md)|  |

### Return type

[**Feed**](Feed.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deleteFeed**
> deleteFeed(id)

Delete feed

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getFeedsApi();
final String id = id_example; // String | Feed ID

try {
    api.deleteFeed(id);
} on DioException catch (e) {
    print('Exception when calling FeedsApi->deleteFeed: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Feed ID |

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **listFeeds**
> FeedListOutputBody listFeeds()

List feeds

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getFeedsApi();

try {
    final response = api.listFeeds();
    print(response);
} on DioException catch (e) {
    print('Exception when calling FeedsApi->listFeeds: $e\n');
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**FeedListOutputBody**](FeedListOutputBody.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **refreshFeed**
> FeedRefreshOutputBody refreshFeed(id)

Refresh feed

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getFeedsApi();
final String id = id_example; // String | Feed ID

try {
    final response = api.refreshFeed(id);
    print(response);
} on DioException catch (e) {
    print('Exception when calling FeedsApi->refreshFeed: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Feed ID |

### Return type

[**FeedRefreshOutputBody**](FeedRefreshOutputBody.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateFeed**
> Feed updateFeed(id, updateFeedInputBody)

Update feed

### Example
```dart
import 'package:liquid2_api/api.dart';

final api = Liquid2Api().getFeedsApi();
final String id = id_example; // String | Feed ID
final UpdateFeedInputBody updateFeedInputBody = ; // UpdateFeedInputBody |

try {
    final response = api.updateFeed(id, updateFeedInputBody);
    print(response);
} on DioException catch (e) {
    print('Exception when calling FeedsApi->updateFeed: $e\n');
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **String**| Feed ID |
 **updateFeedInputBody** | [**UpdateFeedInputBody**](UpdateFeedInputBody.md)|  |

### Return type

[**Feed**](Feed.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)
