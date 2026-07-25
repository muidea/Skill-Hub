package cli

import (
	"context"
	"strings"
	"testing"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/spf13/cobra"
)

func TestUseCommandsDoNotExposeAgentFlag(t *testing.T) {
	for name, command := range map[string]*cobra.Command{
		"use": useCmd, "apply": applyCmd, "status": statusCmd, "remove": removeCmd,
	} {
		t.Run(name, func(t *testing.T) {
			if command.Flags().Lookup("agent") != nil {
				t.Fatalf("%s must not expose --agent", name)
			}
		})
	}
}

func TestRunUseWithOptions_DryRunSelectsExplicitRepository(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		findSkillsByPatternsFn: func(_ context.Context, patterns, repositories []string) ([]spec.SkillMetadata, error) {
			if got, want := repositories, []string{"secondary"}; len(got) != len(want) || got[0] != want[0] {
				t.Fatalf("repository filter = %#v, want %#v", got, want)
			}
			return []spec.SkillMetadata{
				{ID: "demo-skill", Repository: "main", Version: "1.0.0"},
				{ID: "demo-skill", Repository: "secondary", Version: "2.0.0"},
			}, nil
		},
		getSkillDetailFn: func(_ context.Context, skillID, repository string) (*spec.Skill, error) {
			return &spec.Skill{ID: skillID, Repository: repository, Version: "2.0.0", Variables: []spec.Variable{{Name: "region", Default: "cn"}}}, nil
		},
		useSkillFn: func(_ context.Context, _ httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error) {
			t.Fatal("dry-run must not invoke UseSkill")
			return nil, nil
		},
	})
	defer reset()

	summary, err := runUseWithOptions([]string{"demo-skill"}, useOptions{
		Repository:     "secondary",
		DryRun:         true,
		NonInteractive: true,
		ExactInput:     true,
	})
	if err != nil {
		t.Fatalf("runUseWithOptions returned error: %v", err)
	}
	if summary.Planned != 1 || summary.Enabled != 0 || len(summary.Items) != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	item := summary.Items[0]
	if item.Repository != "secondary" || item.Version != "2.0.0" || !item.NeedsVariables {
		t.Fatalf("unexpected dry-run item: %+v", item)
	}
}

func TestRunUseWithOptions_NonInteractiveReportsAmbiguousSource(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		findSkillsByPatternsFn: func(_ context.Context, _ []string, _ []string) ([]spec.SkillMetadata, error) {
			return []spec.SkillMetadata{{ID: "demo-skill", Repository: "main"}, {ID: "demo-skill", Repository: "secondary"}}, nil
		},
	})
	defer reset()

	summary, err := runUseWithOptions([]string{"demo-skill"}, useOptions{NonInteractive: true, ExactInput: true})
	if err == nil || !strings.Contains(err.Error(), "--repo") {
		t.Fatalf("error = %v, want --repo guidance", err)
	}
	if summary.Failed != 1 || len(summary.Items) != 1 || summary.Items[0].Status != "error" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}
