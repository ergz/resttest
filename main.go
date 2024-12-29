package main

import (
	"fmt"
	"io"
	"log"
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
			url:     "https://jsonplaceholder.typicode.com/todos/1",
			name:    "get all todos",
			paths:   []string{"todos"},
			qparams: nil,
			body:    nil,
		},
		{
			method:  "GET",
			url:     "https://jsonplaceholder.typicode.com/todos/101",
			name:    "get all todos 2",
			paths:   nil,
			qparams: nil,
			body:    nil,
		},
		{
			method:  "GET",
			url:     "https://jsonplaceholder.typicode.com/todos/22",
			name:    "get all todos 3",
			paths:   nil,
			qparams: nil,
			body:    nil,
		},
	}

	return model{
		endpoints:    initial_requests,
		requestState: requestState{inProgress: false, lastResponse: globalResponse, spinner: s},
		ui:           uiState{currentFocus: 0, selectedIndex: 1, tabCount: 3},
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
	cursor        int
	respmsg       string
	tabCount      int
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

// TODO: idk if this really needs to be fixed, I think a global response is fine since I am only
// ever going to have one of these
var globalResponse = &apiResponse{}

func (m model) Init() tea.Cmd {
	return nil
}

// TODO: implement parse request, this will read in the data from a file to populate
func parseRequest(line string) {

}

// TODO: implenent a function to construc actual url we hit with the tool
func constructRequest(endpoint string, paths []string, qparams map[string]string) {

}

func makeRequest(endpoint string, method string, gresp *apiResponse) tea.Cmd {

	return func() tea.Msg {

		var resp *http.Response
		var err error

		thisReponse := apiResponse{
			statusCode:   200,
			data:         "",
			error:        nil,
			responseTime: 0,
		}

		log.Println("DEBUG START------------------")
		log.Printf("endpoint: %s", endpoint)
		log.Printf("method: %s", method)
		log.Println("DEBUG START------------------")

		startTime := time.Now()
		if strings.ToLower(method) == "get" {
			resp, err = http.Get(endpoint)
		} else if strings.ToLower(endpoint) == "post" {
			resp, err = http.Get(endpoint)
		}

		if resp == nil {

			log.Printf("the value of the endpoint: %s", endpoint)
			log.Printf("made it this far ---------------")

		}
		defer resp.Body.Close()

		if err != nil {
			thisReponse.statusCode = 404
			thisReponse.error = err
			*gresp = thisReponse
			return thisReponse
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			thisReponse.statusCode = 404
			thisReponse.error = err
			*gresp = thisReponse
			return thisReponse
		}

		thisReponse = apiResponse{
			statusCode:   resp.StatusCode,
			data:         string(body),
			error:        nil,
			responseTime: time.Since(startTime),
		}

		*gresp = thisReponse

		return thisReponse

	}
}

var selectedEndpoint apiRequest

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.ui.currentFocus == 0 {
		// handle keys when user in the endpoints sections
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
			case "tab":
				if m.ui.currentFocus < m.ui.tabCount-1 {
					m.ui.currentFocus++
				} else if m.ui.currentFocus == m.ui.tabCount-1 {
					m.ui.currentFocus = 0
				}
			case "enter", " ":
				if m.requestState.inProgress == false {
					m.requestState.inProgress = true
					selectedEndpoint = m.endpoints[m.ui.cursor]
					return m, tea.Batch(
						makeRequest(selectedEndpoint.url, selectedEndpoint.method, globalResponse),
						m.requestState.spinner.Tick, // add spinner animation
					)
				}

			}
		case apiResponse:
			m.requestState.inProgress = false
			if msg.error != nil {
				m.requestState.lastResponse.error = msg.error
			} else {
				emj := ""
				if msg.statusCode >= 200 && msg.statusCode <= 299 {
					emj = "✅"
				} else {
					emj = "❌"
				}
				m.ui.respmsg = fmt.Sprintf("%s %s Responded with [%d] in %.2fs - %s",
					emj,
					selectedEndpoint.url,
					msg.statusCode,
					msg.responseTime.Seconds(),
					msg.data,
				)
			}
		case spinner.TickMsg:
			var cmd tea.Cmd
			m.requestState.spinner, cmd = m.requestState.spinner.Update(msg)
			return m, cmd

		}

	} else if m.ui.currentFocus == 1 {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "tab":
				if m.ui.currentFocus < m.ui.tabCount-1 {
					m.ui.currentFocus++
				} else if m.ui.currentFocus == m.ui.tabCount-1 {
					m.ui.currentFocus = 0
				}

			}
		}

	} else {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "tab":
				if m.ui.currentFocus < m.ui.tabCount-1 {
					m.ui.currentFocus++
				} else if m.ui.currentFocus == m.ui.tabCount-1 {
					m.ui.currentFocus = 0
				}

			}
		}

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
	for i, choice := range m.endpoints {
		cursor := " "
		if m.ui.cursor == i {
			if m.requestState.inProgress {
				cursor = m.requestState.spinner.View() // show spinner instead of ">"
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

	selectedChoice := m.endpoints[m.ui.cursor]
	renderedNameDetails := selectedChoice.name

	if selectedChoice.paths != nil {
		renderedNameDetails += " [id]"
	}

	leftAppContent = leftAppStyle.Render(s)
	rightAppContent = rightAppStyle.Render(renderedNameDetails)
	reponseAreaContent = responseAreaStyle.Render(m.ui.respmsg)

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
	f, _ := os.Create("debug.log")
	log.SetOutput(f)
	defer f.Close()

	// Then in your code
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("there was an error!")
		fmt.Printf("%s", err)
		os.Exit(1)
	}

}
