package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

const url = "https://google.com"

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	initial_requests := []apiRequest{
		{
			method:  "GET",
			url:     "https://jsonplaceholder.typicode.com/todos/{id}",
			name:    "get all todos",
			paths:   nil,
			qparams: nil,
			body:    nil,
		},
		{
			method:  "GET",
			url:     "https://jsonplaceholder.typicode.com/todos/{id}",
			name:    "get all todos",
			paths:   nil,
			qparams: nil,
			body:    nil,
		},
		{
			method:  "GET",
			url:     "https://jsonplaceholder.typicode.com/todos/{id}",
			name:    "get all todos",
			paths:   nil,
			qparams: nil,
			body:    nil,
		},
	}

	return model{
		endpoints:    initial_requests,
		requestState: requestState{false, nil, s},
		ui:           uiState{currentFocus: 1, selectedIndex: 1, isLoading: false},
	}
}

type apiRequest struct {
	method  string
	url     string
	name    string
	paths   []string
	qparams map[string]string
	body    interface{}
}

type apiResponse struct {
	statusCode   int
	data         string
	error        error
	responseTime time.Duration
}

type uiState struct {
	currentFocus  int
	selectedIndex int
	isLoading     bool
	cursor        int
}

type requestState struct {
	inProgress   bool
	lastResponse *apiResponse
	spinner      spinner.Model
}

type model struct {
	endpoints    []apiRequest
	requestState requestState
	ui           uiState
}

type apiElement struct {
	method      string
	url         string
	name        string
	paths       []string
	queryParams map[string]string
	body        interface{}
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Underline(true).Align(lipgloss.Center).Width(100)
	// Styles for the panes
	leftAppStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("#cf6400")).
			Padding(1).
			Margin(0, 0, 0, 0)

	rightAppStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("#cf6400")).
			Height(1).
			Margin(0)

	responseAreaStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("#cf6400")).
				Width(70).
				Padding(1).
				Margin(0)

	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF7F")).Bold(true)
	itemStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#828282"))
	itemActionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e33939")).Blink(true)
)

func (m model) Init() tea.Cmd {
	return nil
}

func makeRequest(endpoint string, method string, rs *requestState) tea.Cmd {
	startTime := time.Now()
	return func() tea.Msg {

		var resp *http.Response
		var err error

		if method == "get" {
			resp, err = http.Get(endpoint)
		} else if method == "post" {
			resp, err = http.Get(endpoint)
		}

		if err != nil {
			return apiResponse{statusCode: 404, data: "", error: nil, responseTime: 0}
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return apiResponse{statusCode: 404, data: "", error: nil, responseTime: 0}
		}

		resp := apiResponse{
			statusCode:   resp.StatusCode,
			data:         string(body),
			error:        nil,
			responseTime: time.Since(startTime),
		}

		return resp

	}
}

var selectedEndpoint apiRequest

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up", "k":
			if m.ui.cursor > 0 {
				m.ui.cursor--
			}
		case "down", "j":
			if m.ui.cursor < len(m.endpoints)-1 {
				m.ui.cursor++
			}
		case "enter", " ":
			m.requestState.inProgress = true
			selectedEndpoint = m.endpoints[m.ui.cursor]
			response := makeRequest(selectedEndpoint.url, selectedEndpoint.method)
			m.requestState.lastResponse = response
			return m, tea.Batch(
				makeRequest(selectedEndpoint.url, selectedEndpoint.method),
				m.requestState.spinner.Tick, // add spinner animation
			)
		case "tab": // switch bewteen the three sections in the ui
			m.ui.currentFocus = (m.ui.currentFocus + 1) % 3
		}
	case apiResponse:
		m.requestState.inProgress = false
		m.ui.isLoading = false
		if msg.error != nil {
			m.requestState.lastResponse.error = msg.error
		} else {
			emj := ""
			if msg.status >= 200 && msg.status <= 299 {
				emj = "✅"
			} else {
				emj = "❌"
			}
			m.response = fmt.Sprintf("%s %s Responded with [%d] in %.2fs - %s",
				emj,
				selectedEndpoint.url,
				msg.status,
				m.responseTime.Seconds(),
				msg.data,
			)
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, cmd
}

func (m model) View() string {
	s := ""
	// rightVal := ""
	var leftAppContent string
	var rightAppContent string
	var reponseAreaContent string

	var renderedUrl string
	var renderedMethod string
	// var renderedName string
	// create the choices ui
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			if m.isLoading {
				cursor = m.spinner.View() // show spinner instead of ">"
			} else {
				cursor = ">"
			}
			renderedUrl = selectedItemStyle.Render(choice.url)
			renderedMethod = selectedItemStyle.Render(strings.ToUpper(choice.method))
			// renderedName = selectedItemStyle.Render(choice.name)
			// rightVal = choice.name

		} else {
			renderedUrl = itemStyle.Render(choice.url)
			renderedMethod = itemStyle.Render(strings.ToUpper(choice.method))
			// renderedName = itemStyle.Render(choice.name)
		}

		s += fmt.Sprintf("%s %-7s %s\n", cursor, renderedMethod, renderedUrl)

	}

	helpText := "\n\n[enter] to send request and [q] to quit"
	s += helpText

	selectedChoice := m.choices[m.cursor]
	renderedNameDetails := selectedChoice.name

	if selectedChoice.paths != nil {
		renderedNameDetails += " [id]"
	}

	leftAppContent = leftAppStyle.Render(s)
	rightAppContent = rightAppStyle.Render(renderedNameDetails)
	reponseAreaContent = responseAreaStyle.Render(m.response)

	render := lipgloss.JoinVertical(
		lipgloss.Top,
		titleStyle.Render("Resttest"),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftAppContent,
			lipgloss.JoinVertical(lipgloss.Top, rightAppContent, reponseAreaContent)),
	)
	return render
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("there was an error!")
		os.Exit(1)
	}

}
