package ecs_test

import (
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/go-multierror"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/sweep"
)

func init() {
	acctest.RegisterServiceErrorCheckFunc(ecs.EndpointsID, testAccErrorCheckSkipECS)

	resource.AddTestSweepers("aws_ecs_task_definition", &resource.Sweeper{
		Name: "aws_ecs_task_definition",
		F:    testSweepEcsTaskDefinitions,
		Dependencies: []string{
			"aws_ecs_service",
		},
	})
}

func testSweepEcsTaskDefinitions(region string) error {
	client, err := sweep.SharedRegionalSweepClient(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*conns.AWSClient).ECSConn
	var sweeperErrs *multierror.Error

	err = conn.ListTaskDefinitionsPages(&ecs.ListTaskDefinitionsInput{}, func(page *ecs.ListTaskDefinitionsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, taskDefinitionArn := range page.TaskDefinitionArns {
			arn := aws.StringValue(taskDefinitionArn)

			log.Printf("[INFO] Deleting ECS Task Definition: %s", arn)
			_, err := conn.DeregisterTaskDefinition(&ecs.DeregisterTaskDefinitionInput{
				TaskDefinition: aws.String(arn),
			})
			if err != nil {
				sweeperErr := fmt.Errorf("error deleting ECS Task Definition (%s): %w", arn, err)
				log.Printf("[ERROR] %s", sweeperErr)
				sweeperErrs = multierror.Append(sweeperErrs, sweeperErr)
				continue
			}
		}

		return !lastPage
	})
	if sweep.SkipSweepError(err) {
		log.Printf("[WARN] Skipping ECS Task Definitions sweep for %s: %s", region, err)
		return sweeperErrs.ErrorOrNil() // In case we have completed some pages, but had errors
	}
	if err != nil {
		sweeperErrs = multierror.Append(sweeperErrs, fmt.Errorf("error retrieving ECS Task Definitions: %w", err))
	}

	return sweeperErrs.ErrorOrNil()
}

func testAccErrorCheckSkipECS(t *testing.T) resource.ErrorCheckFunc {
	return acctest.ErrorCheckSkipMessagesContaining(t,
		"Unsupported field 'inferenceAccelerators'",
	)
}

func TestAccAWSEcsTaskDefinition_basic(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-basic")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinition(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					acctest.MatchResourceAttrRegionalARN(resourceName, "arn", "ecs", regexp.MustCompile(`task-definition/.+`)),
				),
			},
			{
				Config: testAccAWSEcsTaskDefinitionModified(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					acctest.MatchResourceAttrRegionalARN(resourceName, "arn", "ecs", regexp.MustCompile(`task-definition/.+`)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/2370
func TestAccAWSEcsTaskDefinition_withScratchVolume(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-scratch-volume")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithScratchVolume(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withDockerVolume(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-docker-volume")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithDockerVolumes(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "volume.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "volume.*", map[string]string{
						"name":                                             tdName,
						"docker_volume_configuration.#":                    "1",
						"docker_volume_configuration.0.driver":             "local",
						"docker_volume_configuration.0.scope":              "shared",
						"docker_volume_configuration.0.autoprovision":      "true",
						"docker_volume_configuration.0.driver_opts.%":      "2",
						"docker_volume_configuration.0.driver_opts.device": "tmpfs",
						"docker_volume_configuration.0.driver_opts.uid":    "1000",
						"docker_volume_configuration.0.labels.%":           "2",
						"docker_volume_configuration.0.labels.environment": "test",
						"docker_volume_configuration.0.labels.stack":       "april",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withDockerVolumeMinimalConfig(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-docker-volume")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithDockerVolumesMinimalConfig(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "volume.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "volume.*", map[string]string{
						"name":                          tdName,
						"docker_volume_configuration.#": "1",
						"docker_volume_configuration.0.autoprovision": "true",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withEFSVolumeMinimal(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-efs-volume-min")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithEFSVolumeMinimal(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "volume.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "volume.*", map[string]string{
						"name":                       tdName,
						"efs_volume_configuration.#": "1",
					}),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "volume.*.efs_volume_configuration.0.file_system_id", "aws_efs_file_system.test", "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withEFSVolume(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-efs-volume")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithEFSVolume(tdName, "/home/test"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "volume.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "volume.*", map[string]string{
						"name":                       tdName,
						"efs_volume_configuration.#": "1",
						"efs_volume_configuration.0.root_directory": "/home/test",
					}),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "volume.*.efs_volume_configuration.0.file_system_id", "aws_efs_file_system.test", "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withTransitEncryptionEFSVolume(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-efs-volume")
	resourceName := "aws_ecs_task_definition.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithTransitEncryptionEFSVolume(tdName, "ENABLED", 2999),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "volume.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "volume.*", map[string]string{
						"name":                       tdName,
						"efs_volume_configuration.#": "1",
						"efs_volume_configuration.0.root_directory":          "/home/test",
						"efs_volume_configuration.0.transit_encryption":      "ENABLED",
						"efs_volume_configuration.0.transit_encryption_port": "2999",
					}),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "volume.*.efs_volume_configuration.0.file_system_id", "aws_efs_file_system.test", "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withEFSAccessPoint(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-efs-volume")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithEFSAccessPoint(tdName, "DISABLED"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "volume.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "volume.*", map[string]string{
						"name":                       tdName,
						"efs_volume_configuration.#": "1",
						"efs_volume_configuration.0.root_directory":             "/",
						"efs_volume_configuration.0.transit_encryption":         "ENABLED",
						"efs_volume_configuration.0.transit_encryption_port":    "2999",
						"efs_volume_configuration.0.authorization_config.#":     "1",
						"efs_volume_configuration.0.authorization_config.0.iam": "DISABLED",
					}),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "volume.*.efs_volume_configuration.0.file_system_id", "aws_efs_file_system.test", "id"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "volume.*.efs_volume_configuration.0.authorization_config.0.access_point_id", "aws_efs_access_point.test", "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withFsxWinFileSystem(t *testing.T) {
	var def ecs.TaskDefinition

	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_ecs_task_definition.test"

	if acctest.Partition() == "aws-us-gov" {
		t.Skip("Amazon FSx for Windows File Server volumes for ECS tasks are not supported in GovCloud partition")
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithFsxVolume(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "volume.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "volume.*", map[string]string{
						"name": rName,
						"fsx_windows_file_server_volume_configuration.#":                        "1",
						"fsx_windows_file_server_volume_configuration.0.root_directory":         "\\data",
						"fsx_windows_file_server_volume_configuration.0.authorization_config.#": "1",
					}),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "volume.*.fsx_windows_file_server_volume_configuration.0.file_system_id", "aws_fsx_windows_file_system.test", "id"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "volume.*.fsx_windows_file_server_volume_configuration.0.authorization_config.0.credentials_parameter", "aws_secretsmanager_secret_version.test", "arn"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "volume.*.fsx_windows_file_server_volume_configuration.0.authorization_config.0.domain", "aws_directory_service_directory.test", "name"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withTaskScopedDockerVolume(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-docker-volume")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithTaskScopedDockerVolume(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					testAccCheckAWSTaskDefinitionDockerVolumeConfigurationAutoprovisionNil(&def),
					resource.TestCheckResourceAttr(resourceName, "volume.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/2694
func TestAccAWSEcsTaskDefinition_withEcsService(t *testing.T) {
	var def ecs.TaskDefinition
	var service ecs.Service

	clusterName := sdkacctest.RandomWithPrefix("tf-acc-cluster-with-ecs-service")
	svcName := sdkacctest.RandomWithPrefix("tf-acc-td-with-ecs-service")
	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-ecs-service")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithEcsService(clusterName, svcName, tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					testAccCheckAWSEcsServiceExists("aws_ecs_service.test", &service),
				),
			},
			{
				Config: testAccAWSEcsTaskDefinitionWithEcsServiceModified(clusterName, svcName, tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					testAccCheckAWSEcsServiceExists("aws_ecs_service.test", &service),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withTaskRoleArn(t *testing.T) {
	var def ecs.TaskDefinition

	roleName := sdkacctest.RandomWithPrefix("tf-acc-role-ecs-td-with-task-role-arn")
	policyName := sdkacctest.RandomWithPrefix("tf-acc-policy-ecs-td-with-task-role-arn")
	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-task-role-arn")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithTaskRoleArn(roleName, policyName, tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withNetworkMode(t *testing.T) {
	var def ecs.TaskDefinition

	roleName := sdkacctest.RandomWithPrefix("tf-acc-ecs-td-with-network-mode")
	policyName := sdkacctest.RandomWithPrefix("tf-acc-ecs-td-with-network-mode")
	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-network-mode")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithNetworkMode(roleName, policyName, tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "network_mode", "bridge"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withIPCMode(t *testing.T) {
	var def ecs.TaskDefinition

	roleName := sdkacctest.RandomWithPrefix("tf-acc-ecs-td-with-ipc-mode")
	policyName := sdkacctest.RandomWithPrefix("tf-acc-ecs-td-with-ipc-mode")
	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-ipc-mode")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithIpcMode(roleName, policyName, tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "ipc_mode", "host"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_withPidMode(t *testing.T) {
	var def ecs.TaskDefinition

	roleName := sdkacctest.RandomWithPrefix("tf-acc-ecs-td-with-pid-mode")
	policyName := sdkacctest.RandomWithPrefix("tf-acc-ecs-td-with-pid-mode")
	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-with-pid-mode")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionWithPidMode(roleName, policyName, tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "pid_mode", "host"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_constraint(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-constraint")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinition_constraint(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "placement_constraints.#", "1"),
					testAccCheckAWSTaskDefinitionConstraintsAttrs(&def),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_changeVolumesForcesNewResource(t *testing.T) {
	var before ecs.TaskDefinition
	var after ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-change-vol-forces-new-resource")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinition(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &before),
				),
			},
			{
				Config: testAccAWSEcsTaskDefinitionUpdatedVolume(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &after),
					testAccCheckEcsTaskDefinitionRecreated(t, &before, &after),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform-provider-aws/issues/2336
func TestAccAWSEcsTaskDefinition_arrays(t *testing.T) {
	var conf ecs.TaskDefinition
	resourceName := "aws_ecs_task_definition.test"

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-arrays")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionArrays(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &conf),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_Fargate(t *testing.T) {
	var conf ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-fargate")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionFargate(tdName, `[{"protocol": "tcp", "containerPort": 8000}]`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "requires_compatibilities.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "cpu", "256"),
					resource.TestCheckResourceAttr(resourceName, "memory", "512"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				ExpectNonEmptyPlan: false,
				PlanOnly:           true,
				Config:             testAccAWSEcsTaskDefinitionFargate(tdName, `[{"protocol": "tcp", "containerPort": 8000, "hostPort": 8000}]`),
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_Fargate_ephemeralStorage(t *testing.T) {
	var conf ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-fargate")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionFargateEphemeralStorage(tdName, `[{"protocol": "tcp", "containerPort": 8000}]`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "requires_compatibilities.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "cpu", "256"),
					resource.TestCheckResourceAttr(resourceName, "memory", "512"),
					resource.TestCheckResourceAttr(resourceName, "ephemeral_storage.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ephemeral_storage.0.size_in_gib", "30"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_ExecutionRole(t *testing.T) {
	var conf ecs.TaskDefinition

	roleName := sdkacctest.RandomWithPrefix("tf-acc-role-ecs-td-execution-role")
	policyName := sdkacctest.RandomWithPrefix("tf-acc-policy-ecs-td-execution-role")
	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-execution-role")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionExecutionRole(roleName, policyName, tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &conf),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/3582#issuecomment-286409786
func TestAccAWSEcsTaskDefinition_disappears(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-basic")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinition(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					acctest.CheckResourceDisappears(acctest.Provider, ResourceTaskDefinition(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccAWSEcsTaskDefinition(tdName),
				Check:  resource.TestCheckResourceAttr(resourceName, "revision", "2"), // should get re-created
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_Tags(t *testing.T) {
	var taskDefinition ecs.TaskDefinition
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionConfigTags1(rName, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &taskDefinition),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSEcsTaskDefinitionConfigTags2(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &taskDefinition),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccAWSEcsTaskDefinitionConfigTags1(rName, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &taskDefinition),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_ProxyConfiguration(t *testing.T) {
	var taskDefinition ecs.TaskDefinition
	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_ecs_task_definition.test"

	containerName := "web"
	proxyType := "APPMESH"
	ignoredUid := "1337"
	ignoredGid := "999"
	appPorts := "80"
	proxyIngressPort := "15000"
	proxyEgressPort := "15001"
	egressIgnoredPorts := "5500"
	egressIgnoredIPs := "169.254.170.2,169.254.169.254"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionConfigProxyConfiguration(rName, containerName, proxyType, ignoredUid, ignoredGid, appPorts, proxyIngressPort, proxyEgressPort, egressIgnoredPorts, egressIgnoredIPs),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &taskDefinition),
					testAccCheckAWSEcsTaskDefinitionProxyConfiguration(&taskDefinition, containerName, proxyType, ignoredUid, ignoredGid, appPorts, proxyIngressPort, proxyEgressPort, egressIgnoredPorts, egressIgnoredIPs),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEcsTaskDefinition_inferenceAccelerator(t *testing.T) {
	var def ecs.TaskDefinition

	tdName := sdkacctest.RandomWithPrefix("tf-acc-td-basic")
	resourceName := "aws_ecs_task_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecs.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSEcsTaskDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEcsTaskDefinitionConfigInferenceAccelerator(tdName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcsTaskDefinitionExists(resourceName, &def),
					resource.TestCheckResourceAttr(resourceName, "inference_accelerator.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAWSEcsTaskDefinitionConfigProxyConfiguration(rName string, containerName string, proxyType string,
	ignoredUid string, ignoredGid string, appPorts string, proxyIngressPort string, proxyEgressPort string,
	egressIgnoredPorts string, egressIgnoredIPs string) string {

	return fmt.Sprintf(`
resource "aws_ecs_cluster" "test" {
  name = %q
}

resource "aws_ecs_task_definition" "test" {
  family       = %q
  network_mode = "awsvpc"

  proxy_configuration {
    type           = %q
    container_name = %q
    properties = {
      IgnoredUID         = %q
      IgnoredGID         = %q
      AppPorts           = %q
      ProxyIngressPort   = %q
      ProxyEgressPort    = %q
      EgressIgnoredPorts = %q
      EgressIgnoredIPs   = %q
    }
  }

  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "essential": true,
    "image": "nginx:latest",
    "memory": 128,
    "name": %q
  }
]
DEFINITION
}
`, rName, rName, proxyType, containerName, ignoredUid, ignoredGid, appPorts, proxyIngressPort, proxyEgressPort, egressIgnoredPorts, egressIgnoredIPs, containerName)
}

func testAccCheckAWSEcsTaskDefinitionProxyConfiguration(after *ecs.TaskDefinition, containerName string, proxyType string,
	ignoredUid string, ignoredGid string, appPorts string, proxyIngressPort string, proxyEgressPort string,
	egressIgnoredPorts string, egressIgnoredIPs string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *after.ProxyConfiguration.Type != proxyType {
			return fmt.Errorf("Expected (%s) ProxyConfiguration.Type, got (%s)", proxyType, *after.ProxyConfiguration.Type)
		}

		if *after.ProxyConfiguration.ContainerName != containerName {
			return fmt.Errorf("Expected (%s) ProxyConfiguration.ContainerName, got (%s)", containerName, *after.ProxyConfiguration.ContainerName)
		}

		properties := after.ProxyConfiguration.Properties
		expectedProperties := []string{"IgnoredUID", "IgnoredGID", "AppPorts", "ProxyIngressPort", "ProxyEgressPort", "EgressIgnoredPorts", "EgressIgnoredIPs"}
		if len(properties) != len(expectedProperties) {
			return fmt.Errorf("Expected (%d) ProxyConfiguration.Property count, got (%d)", len(expectedProperties), len(properties))
		}

		propertyLookups := make(map[string]string)
		for _, property := range properties {
			propertyLookups[*property.Name] = *property.Value
		}

		if propertyLookups["IgnoredUID"] != ignoredUid {
			return fmt.Errorf("Expected (%s) ProxyConfiguration.Properties.IgnoredUID, got (%s)", ignoredUid, propertyLookups["IgnoredUID"])
		}

		if propertyLookups["IgnoredGID"] != ignoredGid {
			return fmt.Errorf("Expected (%s) ProxyConfiguration.Properties.IgnoredGID, got (%s)", ignoredGid, propertyLookups["IgnoredGID"])
		}

		if propertyLookups["AppPorts"] != appPorts {
			return fmt.Errorf("Expected (%s) ProxyConfiguration.Properties.AppPorts, got (%s)", appPorts, propertyLookups["AppPorts"])
		}

		if propertyLookups["ProxyIngressPort"] != proxyIngressPort {
			return fmt.Errorf("Expected (%s) ProxyConfiguration.Properties.ProxyIngressPort, got (%s)", proxyIngressPort, propertyLookups["ProxyIngressPort"])
		}

		if propertyLookups["ProxyEgressPort"] != proxyEgressPort {
			return fmt.Errorf("Expected (%s) ProxyConfiguration.Properties.ProxyEgressPort, got (%s)", proxyEgressPort, propertyLookups["ProxyEgressPort"])
		}

		if propertyLookups["EgressIgnoredPorts"] != egressIgnoredPorts {
			return fmt.Errorf("Expected (%s) ProxyConfiguration.Properties.EgressIgnoredPorts, got (%s)", egressIgnoredPorts, propertyLookups["EgressIgnoredPorts"])
		}

		if propertyLookups["EgressIgnoredIPs"] != egressIgnoredIPs {
			return fmt.Errorf("Expected (%s) ProxyConfiguration.Properties.EgressIgnoredIPs, got (%s)", egressIgnoredIPs, propertyLookups["EgressIgnoredIPs"])
		}

		return nil
	}
}

func testAccCheckEcsTaskDefinitionRecreated(t *testing.T,
	before, after *ecs.TaskDefinition) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *before.Revision == *after.Revision {
			t.Fatalf("Expected change of TaskDefinition Revisions, but both were %v", before.Revision)
		}
		return nil
	}
}

func testAccCheckAWSTaskDefinitionConstraintsAttrs(def *ecs.TaskDefinition) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(def.PlacementConstraints) != 1 {
			return fmt.Errorf("Expected (1) placement_constraints, got (%d)", len(def.PlacementConstraints))
		}
		return nil
	}
}

func TestValidateAwsEcsTaskDefinitionContainerDefinitions(t *testing.T) {
	validDefinitions := []string{
		testValidateAwsEcsTaskDefinitionValidContainerDefinitions,
	}
	for _, v := range validDefinitions {
		_, errors := validateAwsEcsTaskDefinitionContainerDefinitions(v, "container_definitions")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid AWS ECS Task Definition Container Definitions: %q", v, errors)
		}
	}

	invalidDefinitions := []string{
		testValidateAwsEcsTaskDefinitionInvalidCommandContainerDefinitions,
	}
	for _, v := range invalidDefinitions {
		_, errors := validateAwsEcsTaskDefinitionContainerDefinitions(v, "container_definitions")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid AWS ECS Task Definition Container Definitions", v)
		}
	}
}

func testAccCheckAWSEcsTaskDefinitionDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).ECSConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ecs_task_definition" {
			continue
		}

		input := ecs.DescribeTaskDefinitionInput{
			TaskDefinition: aws.String(rs.Primary.Attributes["arn"]),
		}

		out, err := conn.DescribeTaskDefinition(&input)

		if err != nil {
			return err
		}

		if out.TaskDefinition != nil && *out.TaskDefinition.Status != ecs.TaskDefinitionStatusInactive {
			return fmt.Errorf("ECS task definition still exists:\n%#v", *out.TaskDefinition)
		}
	}

	return nil
}

func testAccCheckAWSEcsTaskDefinitionExists(name string, def *ecs.TaskDefinition) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).ECSConn

		out, err := conn.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: aws.String(rs.Primary.Attributes["arn"]),
		})
		if err != nil {
			return err
		}
		*def = *out.TaskDefinition

		return nil
	}
}

func testAccCheckAWSTaskDefinitionDockerVolumeConfigurationAutoprovisionNil(def *ecs.TaskDefinition) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(def.Volumes) != 1 {
			return fmt.Errorf("Expected (1) volumes, got (%d)", len(def.Volumes))
		}
		config := def.Volumes[0].DockerVolumeConfiguration
		if config == nil {
			return fmt.Errorf("Expected docker_volume_configuration, got nil")
		}
		if config.Autoprovision != nil {
			return fmt.Errorf("Expected autoprovision to be nil, got %t", *config.Autoprovision)
		}
		return nil
	}
}

func testAccAWSEcsTaskDefinition_constraint(tdName string) string {
	return acctest.ConfigCompose(acctest.ConfigAvailableAZsNoOptIn(), fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family = "%s"

  container_definitions = <<TASK_DEFINITION
[
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"environment": [
			{"name": "VARNAME", "value": "VARVAL"}
		],
		"essential": true,
		"image": "jenkins",
		"links": ["mongodb"],
		"memory": 128,
		"name": "jenkins",
		"portMappings": [
			{
				"containerPort": 80,
				"hostPort": 8080
			}
		]
	},
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"essential": true,
		"image": "mongodb",
		"memory": 128,
		"name": "mongodb",
		"portMappings": [
			{
				"containerPort": 28017,
				"hostPort": 28017
			}
		]
	}
]
TASK_DEFINITION


  volume {
    name      = "jenkins-home"
    host_path = "/ecs/jenkins-home"
  }

  placement_constraints {
    type       = "memberOf"
    expression = "attribute:ecs.availability-zone in [${data.aws_availability_zones.available.names[0]}, ${data.aws_availability_zones.available.names[1]}]"
  }
}
`, tdName))
}

func testAccAWSEcsTaskDefinition(tdName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family = "%s"

  container_definitions = <<TASK_DEFINITION
[
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"environment": [
			{"name": "VARNAME", "value": "VARVAL"}
		],
		"essential": true,
		"image": "jenkins",
		"links": ["mongodb"],
		"memory": 128,
		"name": "jenkins",
		"portMappings": [
			{
				"containerPort": 80,
				"hostPort": 8080
			}
		]
	},
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"essential": true,
		"image": "mongodb",
		"memory": 128,
		"name": "mongodb",
		"portMappings": [
			{
				"containerPort": 28017,
				"hostPort": 28017
			}
		]
	}
]
TASK_DEFINITION


  volume {
    name      = "jenkins-home"
    host_path = "/ecs/jenkins-home"
  }
}
`, tdName)
}

func testAccAWSEcsTaskDefinitionUpdatedVolume(tdName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family = "%s"

  container_definitions = <<TASK_DEFINITION
[
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"environment": [
			{"name": "VARNAME", "value": "VARVAL"}
		],
		"essential": true,
		"image": "jenkins",
		"links": ["mongodb"],
		"memory": 128,
		"name": "jenkins",
		"portMappings": [
			{
				"containerPort": 80,
				"hostPort": 8080
			}
		]
	},
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"essential": true,
		"image": "mongodb",
		"memory": 128,
		"name": "mongodb",
		"portMappings": [
			{
				"containerPort": 28017,
				"hostPort": 28017
			}
		]
	}
]
TASK_DEFINITION


  volume {
    name      = "jenkins-home"
    host_path = "/ecs/jenkins"
  }
}
`, tdName)
}

func testAccAWSEcsTaskDefinitionArrays(tdName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family = "%s"

  container_definitions = <<TASK_DEFINITION
[
    {
      "name": "wordpress",
      "image": "wordpress",
      "essential": true,
      "links": ["container1", "container2", "container3"],
      "portMappings": [
        {"containerPort": 80},
        {"containerPort": 81},
        {"containerPort": 82}
      ],
      "environment": [
        {"name": "VARNAME1", "value": "VARVAL1"},
        {"name": "VARNAME2", "value": "VARVAL2"},
        {"name": "VARNAME3", "value": "VARVAL3"}
      ],
      "extraHosts": [
        {"hostname": "host1", "ipAddress": "127.0.0.1"},
        {"hostname": "host2", "ipAddress": "127.0.0.2"},
        {"hostname": "host3", "ipAddress": "127.0.0.3"}
      ],
      "mountPoints": [
        {"sourceVolume": "vol1", "containerPath": "/vol1"},
        {"sourceVolume": "vol2", "containerPath": "/vol2"},
        {"sourceVolume": "vol3", "containerPath": "/vol3"}
      ],
      "volumesFrom": [
        {"sourceContainer": "container1"},
        {"sourceContainer": "container2"},
        {"sourceContainer": "container3"}
      ],
      "ulimits": [
        {
          "name": "core",
          "softLimit": 10, "hardLimit": 20
        },
        {
          "name": "cpu",
          "softLimit": 10, "hardLimit": 20
        },
        {
          "name": "fsize",
          "softLimit": 10, "hardLimit": 20
        }
      ],
      "linuxParameters": {
        "capabilities": {
          "add": ["AUDIT_CONTROL", "AUDIT_WRITE", "BLOCK_SUSPEND"],
          "drop": ["CHOWN", "IPC_LOCK", "KILL"]
        }
      },
      "devices": [
        {
          "hostPath": "/path1",
          "permissions": ["read", "write", "mknod"]
        },
        {
          "hostPath": "/path2",
          "permissions": ["read", "write"]
        },
        {
          "hostPath": "/path3",
          "permissions": ["read", "mknod"]
        }
      ],
      "dockerSecurityOptions": ["label:one", "label:two", "label:three"],
      "memory": 500,
      "cpu": 10
    },
    {
      "name": "container1",
      "image": "busybox",
      "memory": 100
    },
    {
      "name": "container2",
      "image": "busybox",
      "memory": 100
    },
    {
      "name": "container3",
      "image": "busybox",
      "memory": 100
    }
]
TASK_DEFINITION


  volume {
    name      = "vol1"
    host_path = "/host/vol1"
  }

  volume {
    name      = "vol2"
    host_path = "/host/vol2"
  }

  volume {
    name      = "vol3"
    host_path = "/host/vol3"
  }
}
`, tdName)
}

func testAccAWSEcsTaskDefinitionFargate(tdName, portMappings string) string {
	return fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family                   = "%s"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true,
    "portMappings": %s
  }
]
TASK_DEFINITION
}
`, tdName, portMappings)
}

func testAccAWSEcsTaskDefinitionFargateEphemeralStorage(tdName, portMappings string) string {
	return fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family                   = "%s"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"

  ephemeral_storage {
    size_in_gib = 30
  }

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true,
    "portMappings": %s
  }
]
TASK_DEFINITION
}
`, tdName, portMappings)
}

func testAccAWSEcsTaskDefinitionExecutionRole(roleName, policyName, tdName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test" {
  name = "%s"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_policy" "test" {
  name        = "%s"
  description = "A test policy"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "test" {
  role       = aws_iam_role.test.name
  policy_arn = aws_iam_policy.test.arn
}

resource "aws_ecs_task_definition" "test" {
  family             = "%s"
  execution_role_arn = aws_iam_role.test.arn

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION
}
`, roleName, policyName, tdName)
}

func testAccAWSEcsTaskDefinitionWithScratchVolume(tdName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family = %[1]q

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION


  volume {
    name = %[1]q
  }
}
`, tdName)
}

func testAccAWSEcsTaskDefinitionWithDockerVolumes(tdName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family = %[1]q

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION


  volume {
    name = %[1]q

    docker_volume_configuration {
      driver = "local"
      scope  = "shared"

      driver_opts = {
        device = "tmpfs"
        uid    = "1000"
      }

      labels = {
        environment = "test"
        stack       = "april"
      }

      autoprovision = true
    }
  }
}
`, tdName)
}

func testAccAWSEcsTaskDefinitionWithDockerVolumesMinimalConfig(tdName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family = %[1]q

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION


  volume {
    name = %[1]q

    docker_volume_configuration {
      autoprovision = true
    }
  }
}
`, tdName)
}

func testAccAWSEcsTaskDefinitionWithTaskScopedDockerVolume(tdName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family = %[1]q

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION


  volume {
    name = %[1]q

    docker_volume_configuration {
      scope = "task"
    }
  }
}
`, tdName)
}

func testAccAWSEcsTaskDefinitionWithEFSVolumeMinimal(tdName string) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  creation_token = %[1]q
}

resource "aws_ecs_task_definition" "test" {
  family = %[1]q

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION


  volume {
    name = %[1]q

    efs_volume_configuration {
      file_system_id = aws_efs_file_system.test.id
    }
  }
}
`, tdName)
}

func testAccAWSEcsTaskDefinitionWithEFSVolume(tdName, rDir string) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  creation_token = %[1]q
}

resource "aws_ecs_task_definition" "test" {
  family = %[1]q

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION


  volume {
    name = %[1]q

    efs_volume_configuration {
      file_system_id = aws_efs_file_system.test.id
      root_directory = %[2]q
    }
  }
}
`, tdName, rDir)
}

func testAccAWSEcsTaskDefinitionWithTransitEncryptionEFSVolume(tdName, tEnc string, tEncPort int) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  creation_token = %[1]q
}

resource "aws_ecs_task_definition" "test" {
  family = %[1]q

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION


  volume {
    name = %[1]q

    efs_volume_configuration {
      file_system_id          = aws_efs_file_system.test.id
      root_directory          = "/home/test"
      transit_encryption      = %[2]q
      transit_encryption_port = %[3]d
    }
  }
}
`, tdName, tEnc, tEncPort)
}

func testAccAWSEcsTaskDefinitionWithEFSAccessPoint(tdName, useIam string) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  creation_token = %[1]q
}

resource "aws_efs_access_point" "test" {
  file_system_id = aws_efs_file_system.test.id
  posix_user {
    gid = 1001
    uid = 1001
  }
}

resource "aws_ecs_task_definition" "test" {
  family = %[1]q

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION


  volume {
    name = %[1]q

    efs_volume_configuration {
      file_system_id          = aws_efs_file_system.test.id
      transit_encryption      = "ENABLED"
      transit_encryption_port = 2999
      authorization_config {
        access_point_id = aws_efs_access_point.test.id
        iam             = %[2]q
      }
    }
  }
}
`, tdName, useIam)
}

func testAccAWSEcsTaskDefinitionWithTaskRoleArn(roleName, policyName, tdName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test" {
  name = %[1]q
  path = "/test/"

  assume_role_policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": "sts:AssumeRole",
			"Principal": {
				"Service": "ec2.amazonaws.com"
			},
			"Effect": "Allow",
			"Sid": ""
		}
	]
}
EOF
}

data "aws_partition" "current" {}

resource "aws_iam_role_policy" "test" {
  name = %[2]q
  role = aws_iam_role.test.id

  policy = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Action": [
				"s3:GetBucketLocation",
				"s3:ListAllMyBuckets"
			],
			"Resource": "arn:${data.aws_partition.current.partition}:s3:::*"
		}
	]
}
EOF
}

resource "aws_ecs_task_definition" "test" {
  family        = %[3]q
  task_role_arn = aws_iam_role.test.arn

  container_definitions = <<TASK_DEFINITION
[
	{
		"name": "sleep",
		"image": "busybox",
		"cpu": 10,
		"command": ["sleep","360"],
		"memory": 10,
		"essential": true
	}
]
TASK_DEFINITION


  volume {
    name = %[3]q
  }
}
`, roleName, policyName, tdName)
}

func testAccAWSEcsTaskDefinitionWithIpcMode(roleName, policyName, tdName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test" {
  name = %[1]q
  path = "/test/"

  assume_role_policy = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
	 {
		 "Action": "sts:AssumeRole",
		 "Principal": {
			 "Service": "ec2.amazonaws.com"
		 },
		 "Effect": "Allow",
		 "Sid": ""
	 }
 ]
}
EOF
}

data "aws_partition" "current" {}

resource "aws_iam_role_policy" "test" {
  name = %[2]q
  role = aws_iam_role.test.id

  policy = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
	 {
		 "Effect": "Allow",
		 "Action": [
			 "s3:GetBucketLocation",
			 "s3:ListAllMyBuckets"
		 ],
		 "Resource": "arn:${data.aws_partition.current.partition}:s3:::*"
	 }
 ]
}
 
EOF
}

resource "aws_ecs_task_definition" "test" {
  family        = %[3]q
  task_role_arn = aws_iam_role.test.arn
  network_mode  = "bridge"
  ipc_mode      = "host"

  container_definitions = <<TASK_DEFINITION
[
 {
	 "name": "sleep",
	 "image": "busybox",
	 "cpu": 10,
	 "command": ["sleep","360"],
	 "memory": 10,
	 "essential": true
 }
]
TASK_DEFINITION


  volume {
    name = %[3]q
  }
}
`, roleName, policyName, tdName)
}

func testAccAWSEcsTaskDefinitionWithPidMode(roleName, policyName, tdName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test" {
  name = %[1]q
  path = "/test/"

  assume_role_policy = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
	 {
		 "Action": "sts:AssumeRole",
		 "Principal": {
			 "Service": "ec2.amazonaws.com"
		 },
		 "Effect": "Allow",
		 "Sid": ""
	 }
 ]
}
EOF
}

data "aws_partition" "current" {}

resource "aws_iam_role_policy" "test" {
  name = %[2]q
  role = aws_iam_role.test.id

  policy = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
	 {
		 "Effect": "Allow",
		 "Action": [
			 "s3:GetBucketLocation",
			 "s3:ListAllMyBuckets"
		 ],
		 "Resource": "arn:${data.aws_partition.current.partition}:s3:::*"
	 }
 ]
}
 
EOF
}

resource "aws_ecs_task_definition" "test" {
  family        = %[3]q
  task_role_arn = aws_iam_role.test.arn
  network_mode  = "bridge"
  pid_mode      = "host"

  container_definitions = <<TASK_DEFINITION
[
 {
	 "name": "sleep",
	 "image": "busybox",
	 "cpu": 10,
	 "command": ["sleep","360"],
	 "memory": 10,
	 "essential": true
 }
]
TASK_DEFINITION


  volume {
    name = %[3]q
  }
}
`, roleName, policyName, tdName)
}

func testAccAWSEcsTaskDefinitionWithNetworkMode(roleName, policyName, tdName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test" {
  name = %[1]q
  path = "/test/"

  assume_role_policy = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
	 {
		 "Action": "sts:AssumeRole",
		 "Principal": {
			 "Service": "ec2.amazonaws.com"
		 },
		 "Effect": "Allow",
		 "Sid": ""
	 }
 ]
}
EOF
}

data "aws_partition" "current" {}

resource "aws_iam_role_policy" "test" {
  name = %[2]q
  role = aws_iam_role.test.id

  policy = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
	 {
		 "Effect": "Allow",
		 "Action": [
			 "s3:GetBucketLocation",
			 "s3:ListAllMyBuckets"
		 ],
		 "Resource": "arn:${data.aws_partition.current.partition}:s3:::*"
	 }
 ]
}
 
EOF
}

resource "aws_ecs_task_definition" "test" {
  family        = %[3]q
  task_role_arn = aws_iam_role.test.arn
  network_mode  = "bridge"

  container_definitions = <<TASK_DEFINITION
[
 {
	 "name": "sleep",
	 "image": "busybox",
	 "cpu": 10,
	 "command": ["sleep","360"],
	 "memory": 10,
	 "essential": true
 }
]
TASK_DEFINITION


  volume {
    name = %[3]q
  }
}
`, roleName, policyName, tdName)
}

func testAccAWSEcsTaskDefinitionWithEcsService(clusterName, svcName, tdName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_cluster" "test" {
  name = %[1]q
}

resource "aws_ecs_service" "test" {
  name            = %[2]q
  cluster         = aws_ecs_cluster.test.id
  task_definition = aws_ecs_task_definition.test.arn
  desired_count   = 1
}

resource "aws_ecs_task_definition" "test" {
  family = %[3]q

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION


  volume {
    name = %[3]q
  }
}
`, clusterName, svcName, tdName)
}

func testAccAWSEcsTaskDefinitionWithEcsServiceModified(clusterName, svcName, tdName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_cluster" "test" {
  name = %[1]q
}

resource "aws_ecs_service" "test" {
  name            = %[2]q
  cluster         = aws_ecs_cluster.test.id
  task_definition = aws_ecs_task_definition.test.arn
  desired_count   = 1
}

resource "aws_ecs_task_definition" "test" {
  family = %[3]q

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 20,
    "command": ["sleep","360"],
    "memory": 50,
    "essential": true
  }
]
TASK_DEFINITION


  volume {
    name = %[3]q
  }
}
`, clusterName, svcName, tdName)
}

func testAccAWSEcsTaskDefinitionModified(tdName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family = "%s"

  container_definitions = <<TASK_DEFINITION
[
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"environment": [
			{"name": "VARNAME", "value": "VARVAL"}
		],
		"essential": true,
		"image": "jenkins",
		"links": ["mongodb"],
		"memory": 128,
		"name": "jenkins",
		"portMappings": [
			{
				"containerPort": 80,
				"hostPort": 8080
			}
		]
	},
	{
		"cpu": 20,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"essential": true,
		"image": "mongodb",
		"memory": 128,
		"name": "mongodb",
		"portMappings": [
			{
				"containerPort": 28017,
				"hostPort": 28017
			}
		]
	}
]
TASK_DEFINITION


  volume {
    name      = "jenkins-home"
    host_path = "/ecs/jenkins-home"
  }
}
`, tdName)
}

var testValidateAwsEcsTaskDefinitionValidContainerDefinitions = `
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
`

var testValidateAwsEcsTaskDefinitionInvalidCommandContainerDefinitions = `
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": "sleep 360",
    "memory": 10,
    "essential": true
  }
]
`

func testAccAWSEcsTaskDefinitionConfigTags1(rName, tag1Key, tag1Value string) string {
	return fmt.Sprintf(`
resource "aws_ecs_cluster" "test" {
  name = %q
}

resource "aws_ecs_task_definition" "test" {
  family = %q

  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "essential": true,
    "image": "mongo:latest",
    "memory": 128,
    "name": "mongodb"
  }
]
DEFINITION

  tags = {
    %q = %q
  }
}
`, rName, rName, tag1Key, tag1Value)
}

func testAccAWSEcsTaskDefinitionConfigTags2(rName, tag1Key, tag1Value, tag2Key, tag2Value string) string {
	return fmt.Sprintf(`
resource "aws_ecs_cluster" "test" {
  name = %q
}

resource "aws_ecs_task_definition" "test" {
  family = %q

  container_definitions = <<DEFINITION
[
  {
    "cpu": 128,
    "essential": true,
    "image": "mongo:latest",
    "memory": 128,
    "name": "mongodb"
  }
]
DEFINITION

  tags = {
    %q = %q
    %q = %q
  }
}
`, rName, rName, tag1Key, tag1Value, tag2Key, tag2Value)
}

func testAccAWSEcsTaskDefinitionConfigInferenceAccelerator(tdName string) string {
	return fmt.Sprintf(`
resource "aws_ecs_task_definition" "test" {
  family = "%s"

  container_definitions = <<TASK_DEFINITION
[
	{
		"cpu": 10,
		"command": ["sleep", "10"],
		"entryPoint": ["/"],
		"environment": [
			{"name": "VARNAME", "value": "VARVAL"}
		],
		"essential": true,
		"image": "jenkins",
		"memory": 128,
		"name": "jenkins",
		"portMappings": [
			{
				"containerPort": 80,
				"hostPort": 8080
			}
		],
        "resourceRequirements":[
            {
                "type":"InferenceAccelerator",
                "value":"device_1"
            }
        ]
	}
]
TASK_DEFINITION


  inference_accelerator {
    device_name = "device_1"
    device_type = "eia1.medium"
  }
}
`, tdName)
}

func testAccAWSEcsTaskDefinitionWithFsxVolume(tdName string) string {
	return testAccAwsFsxWindowsFileSystemConfigSubnetIds1() + fmt.Sprintf(`
data "aws_partition" "current" {}

resource "aws_secretsmanager_secret" "test" {
  name                    = %[1]q
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "test" {
  secret_id     = aws_secretsmanager_secret.test.id
  secret_string = jsonencode({ username : "admin", password : aws_directory_service_directory.test.password })
}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = jsonencode({
    "Version" : "2012-10-17",
    "Statement" : [
      {
        "Action" : "sts:AssumeRole",
        "Principal" : {
          "Service" : "ecs-tasks.${data.aws_partition.current.dns_suffix}"
        },
        "Effect" : "Allow",
        "Sid" : ""
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "test" {
  role       = aws_iam_role.test.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role_policy_attachment" "test2" {
  role       = aws_iam_role.test.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/SecretsManagerReadWrite"
}

resource "aws_iam_role_policy_attachment" "test3" {
  role       = aws_iam_role.test.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/AmazonFSxReadOnlyAccess"
}

resource "aws_ecs_task_definition" "test" {
  family             = %[1]q
  execution_role_arn = aws_iam_role.test.arn

  container_definitions = <<TASK_DEFINITION
[
  {
    "name": "sleep",
    "image": "busybox",
    "cpu": 10,
    "command": ["sleep","360"],
    "memory": 10,
    "essential": true
  }
]
TASK_DEFINITION


  volume {
    name = %[1]q

    fsx_windows_file_server_volume_configuration {
      file_system_id = aws_fsx_windows_file_system.test.id
      root_directory = "\\data"

      authorization_config {
        credentials_parameter = aws_secretsmanager_secret_version.test.arn
        domain                = aws_directory_service_directory.test.name
      }
    }
  }

  depends_on = [
    aws_iam_role_policy_attachment.test,
    aws_iam_role_policy_attachment.test2,
    aws_iam_role_policy_attachment.test3
  ]
}
`, tdName)
}

func testAccAWSEcsTaskDefinitionImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}

		return rs.Primary.Attributes["arn"], nil
	}
}

func testAccAwsFsxWindowsFileSystemConfigBase() string {
	return `
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "test1" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]
}

resource "aws_subnet" "test2" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "10.0.2.0/24"
  availability_zone = data.aws_availability_zones.available.names[1]
}

resource "aws_directory_service_directory" "test" {
  edition  = "Standard"
  name     = "corp.notexample.com"
  password = "SuperSecretPassw0rd"
  type     = "MicrosoftAD"

  vpc_settings {
    subnet_ids = [aws_subnet.test1.id, aws_subnet.test2.id]
    vpc_id     = aws_vpc.test.id
  }
}
`
}

func testAccAwsFsxWindowsFileSystemConfigSubnetIds1() string {
	return testAccAwsFsxWindowsFileSystemConfigBase() + `
resource "aws_fsx_windows_file_system" "test" {
  active_directory_id = aws_directory_service_directory.test.id
  skip_final_backup   = true
  storage_capacity    = 32
  subnet_ids          = [aws_subnet.test1.id]
  throughput_capacity = 8
}
`
}
