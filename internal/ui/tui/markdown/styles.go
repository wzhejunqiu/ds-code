package markdown

import (
	"reflect"
	"sync"

	glamour "charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	glamourStyles "charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/wzhejunqiu/ds-code/internal/ui/theme"
)

const (
	mdColorDeepSeek = "#4D6BFE"
	mdColorText     = "#3D3D3D"
	mdColorMuted    = "#8A8A8A"
	mdColorError    = "#C2410C"
)

var (
	mdMu       sync.Mutex
	mdRenderer *glamour.TermRenderer
	mdWidth    int

	codeBlockBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Divider).
				Padding(0, 1)
)

func markdownRenderer(width int) (*glamour.TermRenderer, error) {
	mdMu.Lock()
	defer mdMu.Unlock()
	if mdRenderer != nil && mdWidth == width {
		return mdRenderer, nil
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(chatMarkdownStyles()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}
	mdRenderer = r
	mdWidth = width
	return r, nil
}

func chatMarkdownStyles() ansi.StyleConfig {
	s := glamourStyles.LightStyleConfig
	zero := uint(0)
	s.Document.Margin = &zero

	deepSeek := mdColorDeepSeek
	text := mdColorText
	muted := mdColorMuted

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
		text := mdColorText
		return &ansi.Chroma{
			Text: ansi.StylePrimitive{Color: &text},
		}
	}
	ch := *c
	clearChromaBackgrounds(&ch)
	errColor := mdColorError
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
