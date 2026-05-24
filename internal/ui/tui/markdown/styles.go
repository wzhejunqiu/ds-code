package markdown

import (
	"reflect"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	glamourStyles "github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/wzhejunqiu/ds-code/internal/ui/theme"
	"github.com/muesli/termenv"
)

var (
	mdMu       sync.Mutex
	mdRenderer *glamour.TermRenderer
	mdWidth    int
	mdProfile  termenv.Profile

	codeBlockBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Divider).
				Padding(0, 1)
)

func markdownRenderer(width int) (*glamour.TermRenderer, error) {
	mdMu.Lock()
	defer mdMu.Unlock()
	profile := lipgloss.ColorProfile()
	if mdRenderer != nil && mdWidth == width && mdProfile == profile {
		return mdRenderer, nil
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(chatMarkdownStyles()),
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}
	mdRenderer = r
	mdWidth = width
	mdProfile = profile
	return r, nil
}

func chatMarkdownStyles() ansi.StyleConfig {
	s := glamourStyles.LightStyleConfig
	zero := uint(0)
	s.Document.Margin = &zero

	deepSeek := string(theme.DeepSeek)
	text := string(theme.Text)
	muted := string(theme.Muted)

	s.Heading = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Color:       &deepSeek,
			Bold:        mdBool(true),
		},
	}
	s.H1 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:     &deepSeek,
			Bold:      mdBool(true),
			Underline: mdBool(true),
		},
	}
	s.H2 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:     &deepSeek,
			Bold:      mdBool(true),
			Underline: mdBool(true),
		},
	}
	s.H3 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &deepSeek,
			Bold:  mdBool(true),
		},
	}
	s.H4 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &text,
			Bold:  mdBool(true),
		},
	}
	s.H5 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &text,
			Bold:  mdBool(false),
		},
	}
	s.H6 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &muted,
			Bold:  mdBool(false),
		},
	}
	s.CodeBlock = ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: &text,
			},
			Margin: &zero,
		},
		Chroma: codeBlockChromaStyles(),
	}
	return s
}

func codeBlockChromaStyles() *ansi.Chroma {
	c := glamourStyles.LightStyleConfig.CodeBlock.Chroma
	if c == nil {
		text := string(theme.Text)
		return &ansi.Chroma{
			Text: ansi.StylePrimitive{Color: &text},
		}
	}
	ch := *c
	clearChromaBackgrounds(&ch)
	errColor := string(theme.Error)
	ch.Error = ansi.StylePrimitive{Color: &errColor}
	return &ch
}

func clearChromaBackgrounds(c *ansi.Chroma) {
	v := reflect.ValueOf(c).Elem()
	primType := reflect.TypeOf(ansi.StylePrimitive{})
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Type() != primType {
			continue
		}
		sp := f.Addr().Interface().(*ansi.StylePrimitive)
		sp.BackgroundColor = nil
	}
}

func mdBool(v bool) *bool { return &v }
