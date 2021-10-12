package glue_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glue"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfglue "github.com/hashicorp/terraform-provider-aws/internal/service/glue"
	"github.com/hashicorp/terraform-provider-aws/internal/sweep"
)

func init() {
	resource.AddTestSweepers("aws_glue_connection", &resource.Sweeper{
		Name: "aws_glue_connection",
		F:    sweepConnections,
	})
}

func sweepConnections(region string) error {
	client, err := sweep.SharedRegionalSweepClient(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*conns.AWSClient).GlueConn
	catalogID := client.(*conns.AWSClient).AccountID

	input := &glue.GetConnectionsInput{
		CatalogId: aws.String(catalogID),
	}
	err = conn.GetConnectionsPages(input, func(page *glue.GetConnectionsOutput, lastPage bool) bool {
		if len(page.ConnectionList) == 0 {
			log.Printf("[INFO] No Glue Connections to sweep")
			return false
		}
		for _, connection := range page.ConnectionList {
			name := aws.StringValue(connection.Name)

			log.Printf("[INFO] Deleting Glue Connection: %s", name)
			err := tfglue.DeleteConnection(conn, catalogID, name)
			if err != nil {
				log.Printf("[ERROR] Failed to delete Glue Connection %s: %s", name, err)
			}
		}
		return !lastPage
	})
	if err != nil {
		if sweep.SkipSweepError(err) {
			log.Printf("[WARN] Skipping Glue Connection sweep for %s: %s", region, err)
			return nil
		}
		return fmt.Errorf("Error retrieving Glue Connections: %s", err)
	}

	return nil
}

func TestAccGlueConnection_basic(t *testing.T) {
	var connection glue.Connection

	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_connection.test"

	jdbcConnectionUrl := fmt.Sprintf("jdbc:mysql://%s/testdatabase", acctest.RandomDomainName())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, glue.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionConfig_Required(rName, jdbcConnectionUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(resourceName, &connection),
					acctest.CheckResourceAttrRegionalARN(resourceName, "arn", "glue", fmt.Sprintf("connection/%s", rName)),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.JDBC_CONNECTION_URL", jdbcConnectionUrl),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.PASSWORD", "testpassword"),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.USERNAME", "testusername"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "physical_connection_requirements.#", "0"),
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

func TestAccGlueConnection_mongoDB(t *testing.T) {
	var connection glue.Connection

	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_connection.test"

	connectionUrl := fmt.Sprintf("mongodb://%s:27017/testdatabase", acctest.RandomDomainName())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, glue.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionConfig_MongoDB(rName, connectionUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(resourceName, &connection),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.CONNECTION_URL", connectionUrl),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.USERNAME", "testusername"),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.PASSWORD", "testpassword"),
					resource.TestCheckResourceAttr(resourceName, "connection_type", "MONGODB"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "physical_connection_requirements.#", "0"),
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

func TestAccGlueConnection_kafka(t *testing.T) {
	var connection glue.Connection

	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_connection.test"

	bootstrapServers := fmt.Sprintf("%s:9094,%s:9094", acctest.RandomDomainName(), acctest.RandomDomainName())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, glue.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionConfig_Kafka(rName, bootstrapServers),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(resourceName, &connection),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.KAFKA_BOOTSTRAP_SERVERS", bootstrapServers),
					resource.TestCheckResourceAttr(resourceName, "connection_type", "KAFKA"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "physical_connection_requirements.#", "0"),
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

func TestAccGlueConnection_network(t *testing.T) {
	var connection glue.Connection

	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_connection.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, glue.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionConfig_Network(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(resourceName, &connection),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "connection_type", "NETWORK"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "physical_connection_requirements.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "physical_connection_requirements.0.availability_zone"),
					resource.TestCheckResourceAttr(resourceName, "physical_connection_requirements.0.security_group_id_list.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "physical_connection_requirements.0.subnet_id"),
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

func TestAccGlueConnection_description(t *testing.T) {
	var connection glue.Connection

	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_connection.test"

	jdbcConnectionUrl := fmt.Sprintf("jdbc:mysql://%s/testdatabase", acctest.RandomDomainName())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, glue.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionConfig_Description(rName, jdbcConnectionUrl, "First Description"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(resourceName, &connection),
					resource.TestCheckResourceAttr(resourceName, "description", "First Description"),
				),
			},
			{
				Config: testAccConnectionConfig_Description(rName, jdbcConnectionUrl, "Second Description"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(resourceName, &connection),
					resource.TestCheckResourceAttr(resourceName, "description", "Second Description"),
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

func TestAccGlueConnection_matchCriteria(t *testing.T) {
	var connection glue.Connection

	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_connection.test"

	jdbcConnectionUrl := fmt.Sprintf("jdbc:mysql://%s/testdatabase", acctest.RandomDomainName())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, glue.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionConfig_MatchCriteria_First(rName, jdbcConnectionUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(resourceName, &connection),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.0", "criteria1"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.1", "criteria2"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.2", "criteria3"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.3", "criteria4"),
				),
			},
			{
				Config: testAccConnectionConfig_MatchCriteria_Second(rName, jdbcConnectionUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(resourceName, &connection),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.0", "criteria1"),
				),
			},
			{
				Config: testAccConnectionConfig_MatchCriteria_Third(rName, jdbcConnectionUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(resourceName, &connection),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.0", "criteria2"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.1", "criteria3"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.2", "criteria4"),
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

func TestAccGlueConnection_physicalConnectionRequirements(t *testing.T) {
	var connection glue.Connection

	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_connection.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, glue.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionConfig_PhysicalConnectionRequirements(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(resourceName, &connection),
					resource.TestCheckResourceAttr(resourceName, "connection_properties.%", "3"),
					resource.TestCheckResourceAttrSet(resourceName, "connection_properties.JDBC_CONNECTION_URL"),
					resource.TestCheckResourceAttrSet(resourceName, "connection_properties.PASSWORD"),
					resource.TestCheckResourceAttrSet(resourceName, "connection_properties.USERNAME"),
					resource.TestCheckResourceAttr(resourceName, "match_criteria.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "physical_connection_requirements.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "physical_connection_requirements.0.availability_zone"),
					resource.TestCheckResourceAttr(resourceName, "physical_connection_requirements.0.security_group_id_list.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "physical_connection_requirements.0.subnet_id"),
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

func TestAccGlueConnection_disappears(t *testing.T) {
	var connection glue.Connection

	rName := sdkacctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_connection.test"

	jdbcConnectionUrl := fmt.Sprintf("jdbc:mysql://%s/testdatabase", acctest.RandomDomainName())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, glue.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckConnectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionConfig_Required(rName, jdbcConnectionUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(resourceName, &connection),
					acctest.CheckResourceDisappears(acctest.Provider, tfglue.ResourceConnection(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckConnectionExists(resourceName string, connection *glue.Connection) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Glue Connection ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).GlueConn
		catalogID, connectionName, err := tfglue.DecodeConnectionID(rs.Primary.ID)
		if err != nil {
			return err
		}

		output, err := conn.GetConnection(&glue.GetConnectionInput{
			CatalogId: aws.String(catalogID),
			Name:      aws.String(connectionName),
		})
		if err != nil {
			return err
		}

		if output.Connection == nil {
			return fmt.Errorf("Glue Connection (%s) not found", rs.Primary.ID)
		}

		if aws.StringValue(output.Connection.Name) == connectionName {
			*connection = *output.Connection
			return nil
		}

		return fmt.Errorf("Glue Connection (%s) not found", rs.Primary.ID)
	}
}

func testAccCheckConnectionDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_glue_connection" {
			continue
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).GlueConn
		catalogID, connectionName, err := tfglue.DecodeConnectionID(rs.Primary.ID)
		if err != nil {
			return err
		}

		output, err := conn.GetConnection(&glue.GetConnectionInput{
			CatalogId: aws.String(catalogID),
			Name:      aws.String(connectionName),
		})

		if err != nil {
			if tfawserr.ErrMessageContains(err, glue.ErrCodeEntityNotFoundException, "") {
				return nil
			}

		}

		connection := output.Connection
		if connection != nil && aws.StringValue(connection.Name) == connectionName {
			return fmt.Errorf("Glue Connection %s still exists", rs.Primary.ID)
		}

		return err
	}

	return nil
}

func testAccConnectionConfig_Description(rName, jdbcConnectionUrl, description string) string {
	return fmt.Sprintf(`
resource "aws_glue_connection" "test" {
  name        = %[1]q
  description = %[2]q

  connection_properties = {
    JDBC_CONNECTION_URL = %[3]q
    PASSWORD            = "testpassword"
    USERNAME            = "testusername"
  }

}
`, rName, description, jdbcConnectionUrl)
}

func testAccConnectionConfig_MatchCriteria_First(rName, jdbcConnectionUrl string) string {
	return fmt.Sprintf(`
resource "aws_glue_connection" "test" {
  name = %[1]q

  match_criteria = ["criteria1", "criteria2", "criteria3", "criteria4"]

  connection_properties = {
    JDBC_CONNECTION_URL = %[2]q
    PASSWORD            = "testpassword"
    USERNAME            = "testusername"
  }
}
`, rName, jdbcConnectionUrl)
}

func testAccConnectionConfig_MatchCriteria_Second(rName, jdbcConnectionUrl string) string {
	return fmt.Sprintf(`
resource "aws_glue_connection" "test" {
  name = %[1]q

  match_criteria = ["criteria1"]

  connection_properties = {
    JDBC_CONNECTION_URL = %[2]q
    PASSWORD            = "testpassword"
    USERNAME            = "testusername"
  }
}
`, rName, jdbcConnectionUrl)
}

func testAccConnectionConfig_MatchCriteria_Third(rName, jdbcConnectionUrl string) string {
	return fmt.Sprintf(`
resource "aws_glue_connection" "test" {
  name = "%s"

  match_criteria = ["criteria2", "criteria3", "criteria4"]

  connection_properties = {
    JDBC_CONNECTION_URL = %[2]q
    PASSWORD            = "testpassword"
    USERNAME            = "testusername"
  }
}
`, rName, jdbcConnectionUrl)
}

func testAccConnectionConfig_PhysicalConnectionRequirements(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-glue-connection-base"
  }
}

resource "aws_security_group" "test" {
  name   = "%[1]s"
  vpc_id = aws_vpc.test.id

  ingress {
    from_port = 1
    protocol  = "tcp"
    self      = true
    to_port   = 65535
  }
}

resource "aws_subnet" "test" {
  count = 2

  availability_zone = data.aws_availability_zones.available.names[count.index]
  cidr_block        = "10.0.${count.index}.0/24"
  vpc_id            = aws_vpc.test.id

  tags = {
    Name = "terraform-testacc-glue-connection-base"
  }
}

resource "aws_db_subnet_group" "test" {
  name       = "%[1]s"
  subnet_ids = aws_subnet.test[*].id
}

resource "aws_rds_cluster" "test" {
  cluster_identifier              = "%[1]s"
  database_name                   = "gluedatabase"
  db_cluster_parameter_group_name = "default.aurora-mysql5.7"
  db_subnet_group_name            = aws_db_subnet_group.test.name
  engine                          = "aurora-mysql"
  master_password                 = "gluepassword"
  master_username                 = "glueusername"
  skip_final_snapshot             = true
  vpc_security_group_ids          = [aws_security_group.test.id]
}

resource "aws_rds_cluster_instance" "test" {
  cluster_identifier = aws_rds_cluster.test.id
  engine             = "aurora-mysql"
  identifier         = "%[1]s"
  instance_class     = "db.t2.medium"
}

resource "aws_glue_connection" "test" {
  connection_properties = {
    JDBC_CONNECTION_URL = "jdbc:mysql://${aws_rds_cluster.test.endpoint}/${aws_rds_cluster.test.database_name}"
    PASSWORD            = aws_rds_cluster.test.master_password
    USERNAME            = aws_rds_cluster.test.master_username
  }

  name = "%[1]s"

  physical_connection_requirements {
    availability_zone      = aws_subnet.test[0].availability_zone
    security_group_id_list = [aws_security_group.test.id]
    subnet_id              = aws_subnet.test[0].id
  }
}
`, rName)
}

func testAccConnectionConfig_Required(rName, jdbcConnectionUrl string) string {
	return fmt.Sprintf(`
resource "aws_glue_connection" "test" {
  name = %[1]q

  connection_properties = {
    JDBC_CONNECTION_URL = %[2]q
    PASSWORD            = "testpassword"
    USERNAME            = "testusername"
  }
}
`, rName, jdbcConnectionUrl)
}

func testAccConnectionConfig_MongoDB(rName, connectionUrl string) string {
	return fmt.Sprintf(`
resource "aws_glue_connection" "test" {
  name = %[1]q

  connection_type = "MONGODB"
  connection_properties = {
    CONNECTION_URL = %[2]q
    PASSWORD       = "testpassword"
    USERNAME       = "testusername"
  }
}
`, rName, connectionUrl)
}

func testAccConnectionConfig_Kafka(rName, bootstrapServers string) string {
	return fmt.Sprintf(`
resource "aws_glue_connection" "test" {
  name = %[1]q

  connection_type = "KAFKA"
  connection_properties = {
    KAFKA_BOOTSTRAP_SERVERS = %[2]q
  }
}
`, rName, bootstrapServers)
}

func testAccConnectionConfig_Network(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-glue-connection-network"
  }
}

resource "aws_subnet" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  cidr_block        = "10.0.0.0/24"
  vpc_id            = aws_vpc.test.id

  tags = {
    Name = "terraform-testacc-glue-connection-network"
  }
}

resource "aws_security_group" "test" {
  name   = "%[1]s"
  vpc_id = aws_vpc.test.id

  ingress {
    protocol  = "tcp"
    self      = true
    from_port = 1
    to_port   = 65535
  }
}

resource "aws_glue_connection" "test" {
  connection_type = "NETWORK"
  name            = "%[1]s"

  physical_connection_requirements {
    availability_zone      = aws_subnet.test.availability_zone
    security_group_id_list = [aws_security_group.test.id]
    subnet_id              = aws_subnet.test.id
  }
}
`, rName)
}
