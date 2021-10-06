package kafka

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kafka"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/aws/internal/service/kafka/finder"
	"github.com/hashicorp/terraform-provider-aws/aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	tfkafka "github.com/hashicorp/terraform-provider-aws/internal/service/kafka"
	tfkafka "github.com/hashicorp/terraform-provider-aws/internal/service/kafka"
	tfkafka "github.com/hashicorp/terraform-provider-aws/internal/service/kafka"
	tfkafka "github.com/hashicorp/terraform-provider-aws/internal/service/kafka"
)

func statusClusterState(conn *kafka.Kafka, arn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := tfkafka.FindClusterByARN(conn, arn)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, aws.StringValue(output.State), nil
	}
}

func statusClusterOperationState(conn *kafka.Kafka, arn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := tfkafka.FindClusterOperationByARN(conn, arn)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, aws.StringValue(output.OperationState), nil
	}
}

func statusConfigurationState(conn *kafka.Kafka, arn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := tfkafka.FindConfigurationByARN(conn, arn)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, aws.StringValue(output.State), nil
	}
}
