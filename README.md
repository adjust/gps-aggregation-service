# Multi-Browser Aggregation Service Prototype

The code in here is related to a prototype for aggregating data across browsers with privacy protection. The mechanism is explained [here](https://github.com/WICG/conversion-measurement-api/blob/master/SERVICE.md). Note that the current MPC protocol has some known flaws. This prototype is a proof of concept, and is not the final design.

# How to build

To build the code, you need to install [Bazel](https://bazel.build/) first, and there are detailed instructions in the Go files for running the binaries.

# Main pipelines

There are four main pipelines implemented based on [Apache Beam](https://beam.apache.org/). As instructed in the Go files, you can run the pipelines locally, or use other runner engines such as Google Cloud Dataflow. For the latter, you need to have a Google Cloud project first.

1. `generate_partial_report` simulates the process that the browsers split the conversion data into shares and encrypt with the public keys from helper servers. The input conversion data is CSV in the format `conversion_key,value`, where `conversion_key` is a string and `value` is an integer. `pipeline/test_conversion_data.csv` is an example input conversion file.

2. `exponentiate_conversion_key` applies exponentiation with secrets from both helpers on the encrypted conversion keys for the purpose of MPC.

3. `aggregate_partial_report` calculates the summation of values and count for each encrypted key. Differential privacy is achieved in the results with functions from [Privacy-on-Beam](https://github.com/google/differential-privacy/tree/main/privacy-on-beam).

4. `merge_partial_aggregation` shows an example of how the report origins can obtain the complete aggregation result from the partial results.

# Disclaimer

This is not an officially supported Google product.
