package ec2_test

import (
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/go-multierror"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
	"github.com/hashicorp/terraform-provider-aws/internal/sweep"
)

func init() {
	resource.AddTestSweepers("aws_flow_log", &resource.Sweeper{
		Name: "aws_flow_log",
		F:    sweepFlowLogs,
	})
}

func sweepFlowLogs(region string) error {
	client, err := sweep.SharedRegionalSweepClient(region)
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	conn := client.(*conns.AWSClient).EC2Conn
	var sweeperErrs *multierror.Error

	err = conn.DescribeFlowLogsPages(&ec2.DescribeFlowLogsInput{}, func(page *ec2.DescribeFlowLogsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, flowLog := range page.FlowLogs {
			id := aws.StringValue(flowLog.FlowLogId)

			log.Printf("[INFO] Deleting Flow Log: %s", id)
			_, err := conn.DeleteFlowLogs(&ec2.DeleteFlowLogsInput{
				FlowLogIds: aws.StringSlice([]string{id}),
			})
			if tfawserr.ErrMessageContains(err, "InvalidFlowLogId.NotFound", "") {
				continue
			}
			if err != nil {
				sweeperErr := fmt.Errorf("error deleting Flow Log (%s): %w", id, err)
				log.Printf("[ERROR] %s", sweeperErr)
				sweeperErrs = multierror.Append(sweeperErrs, sweeperErr)
				continue
			}
		}

		return !lastPage
	})
	if sweep.SkipSweepError(err) {
		log.Printf("[WARN] Skipping Flow Logs sweep for %s: %s", region, err)
		return sweeperErrs.ErrorOrNil() // In case we have completed some pages, but had errors
	}
	if err != nil {
		sweeperErrs = multierror.Append(sweeperErrs, fmt.Errorf("error retrieving Flow Logs: %w", err))
	}

	return sweeperErrs.ErrorOrNil()
}

func TestAccEC2FlowLog_vpcID(t *testing.T) {
	var flowLog ec2.FlowLog
	cloudwatchLogGroupResourceName := "aws_cloudwatch_log_group.test"
	iamRoleResourceName := "aws_iam_role.test"
	resourceName := "aws_flow_log.test"
	vpcResourceName := "aws_vpc.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowLogConfig_VPCID(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists(resourceName, &flowLog),
					acctest.MatchResourceAttrRegionalARN(resourceName, "arn", "ec2", regexp.MustCompile(`vpc-flow-log/fl-.+`)),
					testAccCheckFlowLogAttributes(&flowLog),
					resource.TestCheckResourceAttrPair(resourceName, "iam_role_arn", iamRoleResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "log_destination", ""),
					resource.TestCheckResourceAttr(resourceName, "log_destination_type", "cloud-watch-logs"),
					resource.TestCheckResourceAttrPair(resourceName, "log_group_name", cloudwatchLogGroupResourceName, "name"),
					resource.TestCheckResourceAttr(resourceName, "max_aggregation_interval", "600"),
					resource.TestCheckResourceAttr(resourceName, "traffic_type", "ALL"),
					resource.TestCheckResourceAttrPair(resourceName, "vpc_id", vpcResourceName, "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:             testAccFlowLogConfig_LogDestinationType_CloudWatchLogs(rName),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccEC2FlowLog_logFormat(t *testing.T) {
	var flowLog ec2.FlowLog
	resourceName := "aws_flow_log.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	logFormat := "${version} ${vpc-id} ${subnet-id}"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowLogConfig_LogFormat(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists(resourceName, &flowLog),
					testAccCheckFlowLogAttributes(&flowLog),
					resource.TestCheckResourceAttr(resourceName, "log_format", logFormat),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:             testAccFlowLogConfig_LogDestinationType_CloudWatchLogs(rName),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccEC2FlowLog_subnetID(t *testing.T) {
	var flowLog ec2.FlowLog
	cloudwatchLogGroupResourceName := "aws_cloudwatch_log_group.test"
	iamRoleResourceName := "aws_iam_role.test"
	resourceName := "aws_flow_log.test"
	subnetResourceName := "aws_subnet.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowLogConfig_SubnetID(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists(resourceName, &flowLog),
					testAccCheckFlowLogAttributes(&flowLog),
					resource.TestCheckResourceAttrPair(resourceName, "iam_role_arn", iamRoleResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "log_destination", ""),
					resource.TestCheckResourceAttr(resourceName, "log_destination_type", "cloud-watch-logs"),
					resource.TestCheckResourceAttrPair(resourceName, "log_group_name", cloudwatchLogGroupResourceName, "name"),
					resource.TestCheckResourceAttr(resourceName, "max_aggregation_interval", "600"),
					resource.TestCheckResourceAttrPair(resourceName, "subnet_id", subnetResourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "traffic_type", "ALL"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEC2FlowLog_LogDestinationType_cloudWatchLogs(t *testing.T) {
	var flowLog ec2.FlowLog
	cloudwatchLogGroupResourceName := "aws_cloudwatch_log_group.test"
	resourceName := "aws_flow_log.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowLogConfig_LogDestinationType_CloudWatchLogs(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists(resourceName, &flowLog),
					testAccCheckFlowLogAttributes(&flowLog),
					// We automatically trim :* from ARNs if present
					acctest.CheckResourceAttrRegionalARN(resourceName, "log_destination", "logs", fmt.Sprintf("log-group:%s", rName)),
					resource.TestCheckResourceAttr(resourceName, "log_destination_type", "cloud-watch-logs"),
					resource.TestCheckResourceAttrPair(resourceName, "log_group_name", cloudwatchLogGroupResourceName, "name"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEC2FlowLog_LogDestinationType_s3(t *testing.T) {
	var flowLog ec2.FlowLog
	s3ResourceName := "aws_s3_bucket.test"
	resourceName := "aws_flow_log.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowLogConfig_LogDestinationType_S3(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists(resourceName, &flowLog),
					testAccCheckFlowLogAttributes(&flowLog),
					resource.TestCheckResourceAttrPair(resourceName, "log_destination", s3ResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "log_destination_type", "s3"),
					resource.TestCheckResourceAttr(resourceName, "log_group_name", ""),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEC2FlowLog_LogDestinationTypeS3_invalid(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix("tf-acc-test-flow-log-s3-invalid")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccFlowLogConfig_LogDestinationType_S3_Invalid(rName),
				ExpectError: regexp.MustCompile(`(Access Denied for LogDestination|does not exist)`),
			},
		},
	})
}

func TestAccEC2FlowLog_LogDestinationType_maxAggregationInterval(t *testing.T) {
	var flowLog ec2.FlowLog
	resourceName := "aws_flow_log.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowLogConfig_MaxAggregationInterval(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists(resourceName, &flowLog),
					testAccCheckFlowLogAttributes(&flowLog),
					resource.TestCheckResourceAttr(resourceName, "max_aggregation_interval", "60"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEC2FlowLog_tags(t *testing.T) {
	var flowLog ec2.FlowLog
	resourceName := "aws_flow_log.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowLogConfigTags1(rName, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists(resourceName, &flowLog),
					testAccCheckFlowLogAttributes(&flowLog),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccFlowLogConfigTags2(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists(resourceName, &flowLog),
					testAccCheckFlowLogAttributes(&flowLog),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccFlowLogConfigTags1(rName, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists(resourceName, &flowLog),
					testAccCheckFlowLogAttributes(&flowLog),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func TestAccEC2FlowLog_disappears(t *testing.T) {
	var flowLog ec2.FlowLog
	resourceName := "aws_flow_log.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowLogConfig_VPCID(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists(resourceName, &flowLog),
					acctest.CheckResourceDisappears(acctest.Provider, tfec2.ResourceFlowLog(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckFlowLogExists(n string, flowLog *ec2.FlowLog) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Flow Log ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn
		describeOpts := &ec2.DescribeFlowLogsInput{
			FlowLogIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeFlowLogs(describeOpts)
		if err != nil {
			return err
		}

		if len(resp.FlowLogs) > 0 {
			*flowLog = *resp.FlowLogs[0]
			return nil
		}
		return fmt.Errorf("No Flow Logs found for id (%s)", rs.Primary.ID)
	}
}

func testAccCheckFlowLogAttributes(flowLog *ec2.FlowLog) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if flowLog.FlowLogStatus != nil && *flowLog.FlowLogStatus == "ACTIVE" {
			return nil
		}
		if flowLog.FlowLogStatus == nil {
			return fmt.Errorf("Flow Log status is not ACTIVE, is nil")
		} else {
			return fmt.Errorf("Flow Log status is not ACTIVE, got: %s", *flowLog.FlowLogStatus)
		}
	}
}

func testAccCheckFlowLogDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_flow_log" {
			continue
		}

		return nil
	}

	return nil
}

func testAccFlowLogConfigBase(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccFlowLogConfig_LogDestinationType_CloudWatchLogs(rName string) string {
	return testAccFlowLogConfigBase(rName) + fmt.Sprintf(`
data "aws_partition" "current" {}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "ec2.${data.aws_partition.current.dns_suffix}"
        ]
      },
      "Action": [
        "sts:AssumeRole"
      ]
    }
  ]
}
EOF
}

resource "aws_cloudwatch_log_group" "test" {
  name = %[1]q
}

resource "aws_flow_log" "test" {
  iam_role_arn         = aws_iam_role.test.arn
  log_destination      = aws_cloudwatch_log_group.test.arn
  log_destination_type = "cloud-watch-logs"
  traffic_type         = "ALL"
  vpc_id               = aws_vpc.test.id
}
`, rName)
}

func testAccFlowLogConfig_LogDestinationType_S3(rName string) string {
	return testAccFlowLogConfigBase(rName) + fmt.Sprintf(`
resource "aws_s3_bucket" "test" {
  bucket        = %[1]q
  force_destroy = true
}

resource "aws_flow_log" "test" {
  log_destination      = aws_s3_bucket.test.arn
  log_destination_type = "s3"
  traffic_type         = "ALL"
  vpc_id               = aws_vpc.test.id
}
`, rName)
}

func testAccFlowLogConfig_LogDestinationType_S3_Invalid(rName string) string {
	return testAccFlowLogConfigBase(rName) + `
data "aws_partition" "current" {}

resource "aws_flow_log" "test" {
  log_destination      = "arn:${data.aws_partition.current.partition}:s3:::does-not-exist"
  log_destination_type = "s3"
  traffic_type         = "ALL"
  vpc_id               = aws_vpc.test.id
}
`
}

func testAccFlowLogConfig_SubnetID(rName string) string {
	return testAccFlowLogConfigBase(rName) + fmt.Sprintf(`
resource "aws_subnet" "test" {
  cidr_block = "10.0.1.0/24"
  vpc_id     = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

data "aws_partition" "current" {}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "ec2.${data.aws_partition.current.dns_suffix}"
        ]
      },
      "Action": [
        "sts:AssumeRole"
      ]
    }
  ]
}
EOF
}

resource "aws_cloudwatch_log_group" "test" {
  name = %[1]q
}

resource "aws_flow_log" "test" {
  iam_role_arn   = aws_iam_role.test.arn
  log_group_name = aws_cloudwatch_log_group.test.name
  subnet_id      = aws_subnet.test.id
  traffic_type   = "ALL"
}
`, rName)
}

func testAccFlowLogConfig_VPCID(rName string) string {
	return testAccFlowLogConfigBase(rName) + fmt.Sprintf(`
data "aws_partition" "current" {}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "ec2.${data.aws_partition.current.dns_suffix}"
        ]
      },
      "Action": [
        "sts:AssumeRole"
      ]
    }
  ]
}
EOF
}

resource "aws_cloudwatch_log_group" "test" {
  name = %[1]q
}

resource "aws_flow_log" "test" {
  iam_role_arn   = aws_iam_role.test.arn
  log_group_name = aws_cloudwatch_log_group.test.name
  traffic_type   = "ALL"
  vpc_id         = aws_vpc.test.id
}
`, rName)
}

func testAccFlowLogConfig_LogFormat(rName string) string {
	return testAccFlowLogConfigBase(rName) + fmt.Sprintf(`
data "aws_partition" "current" {}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "ec2.${data.aws_partition.current.dns_suffix}"
        ]
      },
      "Action": [
        "sts:AssumeRole"
      ]
    }
  ]
}
EOF
}

resource "aws_cloudwatch_log_group" "test" {
  name = %[1]q
}

resource "aws_s3_bucket" "test" {
  bucket        = %[1]q
  force_destroy = true
}

resource "aws_flow_log" "test" {
  log_destination      = aws_s3_bucket.test.arn
  log_destination_type = "s3"
  traffic_type         = "ALL"
  vpc_id               = aws_vpc.test.id
  log_format           = "$${version} $${vpc-id} $${subnet-id}"
}
`, rName)
}

func testAccFlowLogConfigTags1(rName, tagKey1, tagValue1 string) string {
	return testAccFlowLogConfigBase(rName) + fmt.Sprintf(`
data "aws_partition" "current" {}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "ec2.${data.aws_partition.current.dns_suffix}"
        ]
      },
      "Action": [
        "sts:AssumeRole"
      ]
    }
  ]
}
EOF
}

resource "aws_cloudwatch_log_group" "test" {
  name = %[1]q
}

resource "aws_flow_log" "test" {
  iam_role_arn   = aws_iam_role.test.arn
  log_group_name = aws_cloudwatch_log_group.test.name
  traffic_type   = "ALL"
  vpc_id         = aws_vpc.test.id

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey1, tagValue1)
}

func testAccFlowLogConfigTags2(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return testAccFlowLogConfigBase(rName) + fmt.Sprintf(`
data "aws_partition" "current" {}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "ec2.${data.aws_partition.current.dns_suffix}"
        ]
      },
      "Action": [
        "sts:AssumeRole"
      ]
    }
  ]
}
EOF
}

resource "aws_cloudwatch_log_group" "test" {
  name = %[1]q
}

resource "aws_flow_log" "test" {
  iam_role_arn   = aws_iam_role.test.arn
  log_group_name = aws_cloudwatch_log_group.test.name
  traffic_type   = "ALL"
  vpc_id         = aws_vpc.test.id

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2)
}

func testAccFlowLogConfig_MaxAggregationInterval(rName string) string {
	return testAccFlowLogConfigBase(rName) + fmt.Sprintf(`
data "aws_partition" "current" {}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "ec2.${data.aws_partition.current.dns_suffix}"
        ]
      },
      "Action": [
        "sts:AssumeRole"
      ]
    }
  ]
}
EOF
}

resource "aws_cloudwatch_log_group" "test" {
  name = %[1]q
}

resource "aws_flow_log" "test" {
  iam_role_arn   = aws_iam_role.test.arn
  log_group_name = aws_cloudwatch_log_group.test.name
  traffic_type   = "ALL"
  vpc_id         = aws_vpc.test.id

  max_aggregation_interval = 60
}
`, rName)
}
