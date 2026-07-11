import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';


/// tests for JobsApi
void main() {
  final instance = Liquid2Api().getJobsApi();

  group(JobsApi, () {
    // Get job
    //
    //Future<Job> getJob(String id) async
    test('test getJob', () async {
      // TODO
    });

    // List jobs
    //
    //Future<JobList> listJobs({ String status, String kind, int limit }) async
    test('test listJobs', () async {
      // TODO
    });

  });
}
