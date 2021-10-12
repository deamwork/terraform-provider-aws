package waf_test

import (
	"fmt"
	"log"
	"regexp"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/go-multierror"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfwaf "github.com/hashicorp/terraform-provider-aws/internal/service/waf"
	"github.com/hashicorp/terraform-provider-aws/internal/sweep"
)

func init() {
	resource.AddTestSweepers("aws_waf_rule_group", &resource.Sweeper{
		Name: "aws_waf_rule_group",
		F:    testSweepWafRuleGroups,
		Dependencies: []string{
			"aws_waf_web_acl",
		},
	})
}

func testSweepWafRuleGroups(region string) error {
	client, err := sweep.SharedRegionalSweepClient(region)

	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}

	conn := client.(*conns.AWSClient).WAFConn
	sweepResources := make([]*sweep.SweepResource, 0)
	var errs *multierror.Error
	var g multierror.Group
	var mutex = &sync.Mutex{}

	input := &waf.ListRuleGroupsInput{}

	err = tfwaf.ListRuleGroupsPages(conn, input, func(page *waf.ListRuleGroupsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, ruleGroup := range page.RuleGroups {
			r := ResourceRuleGroup()
			d := r.Data(nil)

			id := aws.StringValue(ruleGroup.RuleGroupId)
			d.SetId(id)

			// read concurrently and gather errors
			g.Go(func() error {
				// Need to Read first to fill in activated_rule attribute
				err := r.Read(d, client)

				if err != nil {
					sweeperErr := fmt.Errorf("error reading WAF Rule Group (%s): %w", id, err)
					log.Printf("[ERROR] %s", sweeperErr)
					return sweeperErr
				}

				// In case it was already deleted
				if d.Id() == "" {
					return nil
				}

				mutex.Lock()
				defer mutex.Unlock()
				sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))

				return nil
			})
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error listing WAF Rule Group for %s: %w", region, err))
	}

	if err = g.Wait().ErrorOrNil(); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error concurrently reading WAF Rule Groups: %w", err))
	}

	if err = sweep.SweepOrchestrator(sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping WAF Rule Group for %s: %w", region, err))
	}

	if sweep.SkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping WAF Rule Group sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}

func TestAccAWSWafRuleGroup_basic(t *testing.T) {
	var rule waf.Rule
	var group waf.RuleGroup
	var idx int

	ruleName := fmt.Sprintf("tfacc%s", sdkacctest.RandString(5))
	groupName := fmt.Sprintf("tfacc%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_rule_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckAWSWaf(t) },
		ErrorCheck:   acctest.ErrorCheck(t, waf.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSWafRuleGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRuleGroupConfig(ruleName, groupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRuleExists("aws_waf_rule.test", &rule),
					testAccCheckAWSWafRuleGroupExists(resourceName, &group),
					resource.TestCheckResourceAttr(resourceName, "name", groupName),
					resource.TestCheckResourceAttr(resourceName, "activated_rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "metric_name", groupName),
					computeActivatedRuleWithRuleId(&rule, "COUNT", 50, &idx),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "activated_rule.*", map[string]string{
						"action.0.type": "COUNT",
						"priority":      "50",
						"type":          waf.WafRuleTypeRegular,
					}),
					acctest.MatchResourceAttrGlobalARN(resourceName, "arn", "waf", regexp.MustCompile(`rulegroup/.+`)),
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

func TestAccAWSWafRuleGroup_changeNameForceNew(t *testing.T) {
	var before, after waf.RuleGroup

	ruleName := fmt.Sprintf("tfacc%s", sdkacctest.RandString(5))
	groupName := fmt.Sprintf("tfacc%s", sdkacctest.RandString(5))
	newGroupName := fmt.Sprintf("tfacc%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_rule_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckAWSWaf(t) },
		ErrorCheck:   acctest.ErrorCheck(t, waf.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSWafRuleGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRuleGroupConfig(ruleName, groupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRuleGroupExists(resourceName, &before),
					resource.TestCheckResourceAttr(resourceName, "name", groupName),
					resource.TestCheckResourceAttr(resourceName, "activated_rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "metric_name", groupName),
				),
			},
			{
				Config: testAccAWSWafRuleGroupConfig(ruleName, newGroupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRuleGroupExists(resourceName, &after),
					resource.TestCheckResourceAttr(resourceName, "name", newGroupName),
					resource.TestCheckResourceAttr(resourceName, "activated_rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "metric_name", newGroupName),
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

func TestAccAWSWafRuleGroup_disappears(t *testing.T) {
	var group waf.RuleGroup
	ruleName := fmt.Sprintf("tfacc%s", sdkacctest.RandString(5))
	groupName := fmt.Sprintf("tfacc%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_rule_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckAWSWaf(t) },
		ErrorCheck:   acctest.ErrorCheck(t, waf.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSWafRuleGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRuleGroupConfig(ruleName, groupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRuleGroupExists(resourceName, &group),
					testAccCheckAWSWafRuleGroupDisappears(&group),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSWafRuleGroup_changeActivatedRules(t *testing.T) {
	var rule0, rule1, rule2, rule3 waf.Rule
	var groupBefore, groupAfter waf.RuleGroup
	var idx0, idx1, idx2, idx3 int

	groupName := fmt.Sprintf("tfacc%s", sdkacctest.RandString(5))
	ruleName1 := fmt.Sprintf("tfacc%s", sdkacctest.RandString(5))
	ruleName2 := fmt.Sprintf("tfacc%s", sdkacctest.RandString(5))
	ruleName3 := fmt.Sprintf("tfacc%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_rule_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckAWSWaf(t) },
		ErrorCheck:   acctest.ErrorCheck(t, waf.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSWafRuleGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRuleGroupConfig(ruleName1, groupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafRuleExists("aws_waf_rule.test", &rule0),
					testAccCheckAWSWafRuleGroupExists(resourceName, &groupBefore),
					resource.TestCheckResourceAttr(resourceName, "name", groupName),
					resource.TestCheckResourceAttr(resourceName, "activated_rule.#", "1"),
					computeActivatedRuleWithRuleId(&rule0, "COUNT", 50, &idx0),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "activated_rule.*", map[string]string{
						"action.0.type": "COUNT",
						"priority":      "50",
						"type":          waf.WafRuleTypeRegular,
					}),
				),
			},
			{
				Config: testAccAWSWafRuleGroupConfig_changeActivatedRules(ruleName1, ruleName2, ruleName3, groupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", groupName),
					resource.TestCheckResourceAttr(resourceName, "activated_rule.#", "3"),
					testAccCheckAWSWafRuleGroupExists(resourceName, &groupAfter),

					testAccCheckAWSWafRuleExists("aws_waf_rule.test", &rule1),
					computeActivatedRuleWithRuleId(&rule1, "BLOCK", 10, &idx1),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "activated_rule.*", map[string]string{
						"action.0.type": "BLOCK",
						"priority":      "10",
						"type":          waf.WafRuleTypeRegular,
					}),

					testAccCheckAWSWafRuleExists("aws_waf_rule.test2", &rule2),
					computeActivatedRuleWithRuleId(&rule2, "COUNT", 1, &idx2),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "activated_rule.*", map[string]string{
						"action.0.type": "COUNT",
						"priority":      "1",
						"type":          waf.WafRuleTypeRegular,
					}),

					testAccCheckAWSWafRuleExists("aws_waf_rule.test3", &rule3),
					computeActivatedRuleWithRuleId(&rule3, "BLOCK", 15, &idx3),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "activated_rule.*", map[string]string{
						"action.0.type": "BLOCK",
						"priority":      "15",
						"type":          waf.WafRuleTypeRegular,
					}),
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

// computeActivatedRuleWithRuleId calculates index
// which isn't static because ruleId is generated as part of the test
func computeActivatedRuleWithRuleId(rule *waf.Rule, actionType string, priority int, idx *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ruleResource := ResourceRuleGroup().Schema["activated_rule"].Elem.(*schema.Resource)

		m := map[string]interface{}{
			"action": []interface{}{
				map[string]interface{}{
					"type": actionType,
				},
			},
			"priority": priority,
			"rule_id":  *rule.RuleId,
			"type":     waf.WafRuleTypeRegular,
		}

		f := schema.HashResource(ruleResource)
		*idx = f(m)

		return nil
	}
}

func TestAccAWSWafRuleGroup_Tags(t *testing.T) {
	var group waf.RuleGroup
	groupName := fmt.Sprintf("test%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_rule_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckAWSWaf(t) },
		ErrorCheck:   acctest.ErrorCheck(t, waf.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSWafWebAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRuleGroupConfigTags1(groupName, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRuleGroupExists(resourceName, &group),
					resource.TestCheckResourceAttr(resourceName, "name", groupName),
					resource.TestCheckResourceAttr(resourceName, "activated_rule.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				Config: testAccAWSWafRuleGroupConfigTags2(groupName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRuleGroupExists(resourceName, &group),
					resource.TestCheckResourceAttr(resourceName, "name", groupName),
					resource.TestCheckResourceAttr(resourceName, "activated_rule.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccAWSWafRuleGroupConfigTags1(groupName, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRuleGroupExists(resourceName, &group),
					resource.TestCheckResourceAttr(resourceName, "name", groupName),
					resource.TestCheckResourceAttr(resourceName, "activated_rule.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
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

func TestAccAWSWafRuleGroup_noActivatedRules(t *testing.T) {
	var group waf.RuleGroup
	groupName := fmt.Sprintf("test%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_rule_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckAWSWaf(t) },
		ErrorCheck:   acctest.ErrorCheck(t, waf.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAWSWafRuleGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRuleGroupConfig_noActivatedRules(groupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafRuleGroupExists(resourceName, &group),
					resource.TestCheckResourceAttr(resourceName, "name", groupName),
					resource.TestCheckResourceAttr(resourceName, "activated_rule.#", "0"),
				),
			},
		},
	})
}

func testAccCheckAWSWafRuleGroupDisappears(group *waf.RuleGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).WAFConn

		rResp, err := conn.ListActivatedRulesInRuleGroup(&waf.ListActivatedRulesInRuleGroupInput{
			RuleGroupId: group.RuleGroupId,
		})
		if err != nil {
			return fmt.Errorf("error listing activated rules in WAF Rule Group (%s): %s", aws.StringValue(group.RuleGroupId), err)
		}

		wr := newWafRetryer(conn)
		_, err = wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateRuleGroupInput{
				ChangeToken: token,
				RuleGroupId: group.RuleGroupId,
			}

			for _, rule := range rResp.ActivatedRules {
				rule := &waf.RuleGroupUpdate{
					Action:        aws.String("DELETE"),
					ActivatedRule: rule,
				}
				req.Updates = append(req.Updates, rule)
			}

			return conn.UpdateRuleGroup(req)
		})
		if err != nil {
			return fmt.Errorf("Error Updating WAF Rule Group: %s", err)
		}

		_, err = wr.RetryWithToken(func(token *string) (interface{}, error) {
			opts := &waf.DeleteRuleGroupInput{
				ChangeToken: token,
				RuleGroupId: group.RuleGroupId,
			}
			return conn.DeleteRuleGroup(opts)
		})
		if err != nil {
			return fmt.Errorf("Error Deleting WAF Rule Group: %s", err)
		}
		return nil
	}
}

func testAccCheckAWSWafRuleGroupDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_waf_rule_group" {
			continue
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).WAFConn
		resp, err := conn.GetRuleGroup(&waf.GetRuleGroupInput{
			RuleGroupId: aws.String(rs.Primary.ID),
		})

		if err == nil {
			if *resp.RuleGroup.RuleGroupId == rs.Primary.ID {
				return fmt.Errorf("WAF Rule Group %s still exists", rs.Primary.ID)
			}
		}

		if tfawserr.ErrMessageContains(err, waf.ErrCodeNonexistentItemException, "") {
			return nil
		}

		return err
	}

	return nil
}

func testAccCheckAWSWafRuleGroupExists(n string, group *waf.RuleGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF Rule Group ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).WAFConn
		resp, err := conn.GetRuleGroup(&waf.GetRuleGroupInput{
			RuleGroupId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if *resp.RuleGroup.RuleGroupId == rs.Primary.ID {
			*group = *resp.RuleGroup
			return nil
		}

		return fmt.Errorf("WAF Rule Group (%s) not found", rs.Primary.ID)
	}
}

func testAccAWSWafRuleGroupConfig(ruleName, groupName string) string {
	return fmt.Sprintf(`
resource "aws_waf_rule" "test" {
  name        = "%[1]s"
  metric_name = "%[1]s"
}

resource "aws_waf_rule_group" "test" {
  name        = "%[2]s"
  metric_name = "%[2]s"

  activated_rule {
    action {
      type = "COUNT"
    }

    priority = 50
    rule_id  = aws_waf_rule.test.id
  }
}
`, ruleName, groupName)
}

func testAccAWSWafRuleGroupConfig_changeActivatedRules(ruleName1, ruleName2, ruleName3, groupName string) string {
	return fmt.Sprintf(`
resource "aws_waf_rule" "test" {
  name        = "%[1]s"
  metric_name = "%[1]s"
}

resource "aws_waf_rule" "test2" {
  name        = "%[2]s"
  metric_name = "%[2]s"
}

resource "aws_waf_rule" "test3" {
  name        = "%[3]s"
  metric_name = "%[3]s"
}

resource "aws_waf_rule_group" "test" {
  name        = "%[4]s"
  metric_name = "%[4]s"

  activated_rule {
    action {
      type = "BLOCK"
    }

    priority = 10
    rule_id  = aws_waf_rule.test.id
  }

  activated_rule {
    action {
      type = "COUNT"
    }

    priority = 1
    rule_id  = aws_waf_rule.test2.id
  }

  activated_rule {
    action {
      type = "BLOCK"
    }

    priority = 15
    rule_id  = aws_waf_rule.test3.id
  }
}
`, ruleName1, ruleName2, ruleName3, groupName)
}

func testAccAWSWafRuleGroupConfig_noActivatedRules(groupName string) string {
	return fmt.Sprintf(`
resource "aws_waf_rule_group" "test" {
  name        = "%[1]s"
  metric_name = "%[1]s"
}
`, groupName)
}

func testAccAWSWafRuleGroupConfigTags1(gName, tag1Key, tag1Value string) string {
	return fmt.Sprintf(`
resource "aws_waf_rule_group" "test" {
  name        = "%[1]s"
  metric_name = "%[1]s"

  tags = {
    %q = %q
  }
}
`, gName, tag1Key, tag1Value)
}

func testAccAWSWafRuleGroupConfigTags2(gName, tag1Key, tag1Value, tag2Key, tag2Value string) string {
	return fmt.Sprintf(`
resource "aws_waf_rule_group" "test" {
  name        = "%[1]s"
  metric_name = "%[1]s"

  tags = {
    %q = %q
    %q = %q
  }
}
`, gName, tag1Key, tag1Value, tag2Key, tag2Value)
}
