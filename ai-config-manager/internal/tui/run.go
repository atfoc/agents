package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/atfoc/agents/ai-config-manager/internal/installer"
)

// Run shows the interactive picker for res and tgt and reports what the
// user did.
//
// It passes no program options: no alternate screen, no mouse. Model.View
// never sets tea.View.AltScreen, which is what makes the program render
// inline, in the caller's own terminal, rather than taking it over.
func Run(res installer.Result, tgt installer.TargetResult) (Outcome, error) {
	m := New(res, tgt, installer.ApplyItem, installer.RemoveItem)

	finalModel, err := tea.NewProgram(m).Run()
	if err != nil {
		return Outcome{}, fmt.Errorf("run interactive picker: %w", err)
	}

	final, ok := finalModel.(Model)
	if !ok {
		return Outcome{}, fmt.Errorf("run interactive picker: unexpected model type %T", finalModel)
	}
	return final.Outcome(), nil
}
