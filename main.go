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
	"github.com/go-yaml/yaml"
)

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	requests, err := loadAPIConfig("./user/endpoints.yaml")
	if err != nil {
		fmt.Errorf("error %w", err)
	}

	return model{
		endpoints:    requests.Endpoints,
		requestState: requestState{inProgress: false, lastResponse: globalResponse, spinner: s},
		ui:           uiState{currentFocus: 0, respmsg: "", selectedIndex: 1, tabCount: 3},
	}
}

type apiConfig struct {
	URLs      map[string]string `yaml:"urls"`
	Endpoints []apiRequest      `yaml:"endpoints"`
}

type apiRequest struct {
	Name    string `yaml:"name"`
	Method  string `yaml:"method"`
	BaseURL string `yaml:"baseURL"`
	Path    string `yaml:"path"`
	fullURL string
	Body    interface{}
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
	width         int
	height        int
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
	titleStyle = lipgloss.NewStyle().Bold(true).Underline(true).Align(lipgloss.Center)

	defaultBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#6e37a5"))

	focusedBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00ff7f"))

	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF7F")).Bold(true)
	itemStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#828282"))
	itemActionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e33939")).Blink(true)
)

// TODO: idk if this really needs to be fixed, I think a global response is fine since I am only
// ever going to have one of these
var globalResponse = &apiResponse{}

func loadAPIConfig(path string) (*apiConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading the config file %w", err)
	}

	var cfg apiConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error trying to parse yaml file, %w", err)
	}

	// create the full url for each endpoint
	for i := range cfg.Endpoints {
		cfg.Endpoints[i].fullURL = constructRequest(cfg.Endpoints[i])
	}

	return &cfg, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

// TODO: implement parse request, this will read in the data from a file to populate
func parseRequest(line string) {

}

// TODO: implenent a function to construc actual url we hit with the tool
func constructRequest(reqConfig apiRequest) string {
	fullURL := fmt.Sprintf("%s/%s", reqConfig.BaseURL, reqConfig.Path)

	return fullURL
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
		log.Println("")
		log.Printf("endpoint: %s", endpoint)
		log.Printf("method: %s", method)
		log.Println("")
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
						makeRequest(selectedEndpoint.fullURL, selectedEndpoint.Method, globalResponse),
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
					selectedEndpoint.fullURL,
					msg.statusCode,
					msg.responseTime.Seconds(),
					msg.data,
				)
			}
		case spinner.TickMsg:
			var cmd tea.Cmd
			m.requestState.spinner, cmd = m.requestState.spinner.Update(msg)
			return m, cmd

		case tea.WindowSizeMsg:
			m.ui.width = msg.Width
			m.ui.height = msg.Height
			log.Printf("the size of the window has changed: %d X %d", m.ui.width, m.ui.height)
			return m, nil

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
		case tea.WindowSizeMsg:
			m.ui.width = msg.Width
			m.ui.height = msg.Height
			return m, nil
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
		case tea.WindowSizeMsg:
			m.ui.width = msg.Width
			m.ui.height = msg.Height
			return m, nil
		}

	}

	return m, cmd
}

func (m model) View() string {

	mainHeight := m.ui.height - 3
	leftWidth := m.ui.width/2 - 2
	rightWidth := m.ui.width - leftWidth - 4

	var panel0, panel1, panel2 string
	var panel0_content strings.Builder

	var renderedUrl string
	var renderedMethod string

	for i, choice := range m.endpoints {
		cursor := " "
		if m.ui.cursor == i {
			if m.requestState.inProgress {
				cursor = m.requestState.spinner.View() // show spinner instead of ">"
			} else {
				cursor = "→"
			}
			renderedUrl = selectedItemStyle.Render(choice.fullURL)
			renderedMethod = selectedItemStyle.Render(strings.ToUpper(choice.Method))
			// renderedName = selectedItemStyle.Render(choice.name)
			// rightVal = choice.name

		} else {
			renderedUrl = itemStyle.Render(choice.fullURL)
			renderedMethod = itemStyle.Render(strings.ToUpper(choice.Method))
			// renderedName = itemStyle.Render(choice.name)
		}

		panel0_content.WriteString(fmt.Sprintf("%s %-7s %s\n", cursor, renderedMethod, renderedUrl))

	}

	helpText := "\n\n[enter] to send request and [q] to quit"
	panel0_content.WriteString(helpText)

	if m.ui.currentFocus == 0 {
		panel0 = focusedBorderStyle.
			Height(mainHeight).
			Width(leftWidth).
			Render(panel0_content.String())
	} else {
		panel0 = defaultBorderStyle.
			Height(mainHeight).
			Width(leftWidth).
			Render(panel0_content.String())
	}

	selectedEndpoint := m.endpoints[m.ui.cursor]
	if m.ui.currentFocus == 1 {
		panel1 = focusedBorderStyle.
			Height(1).
			Width(rightWidth).
			Render(selectedEndpoint.Name)
	} else {
		panel1 = defaultBorderStyle.
			Height(1).
			Width(rightWidth).
			Render(selectedEndpoint.Name)
	}

	if m.ui.respmsg == "" {
		m.ui.respmsg = "run an endpoint to view response"
	}
	if m.ui.currentFocus == 2 {
		panel2 = focusedBorderStyle.
			Height(mainHeight - 3).
			Width(rightWidth).
			Render(m.ui.respmsg)
	} else {
		panel2 = defaultBorderStyle.
			Height(mainHeight - 3).
			Width(rightWidth).
			Render(m.ui.respmsg)
	}

	render := lipgloss.JoinVertical(
		lipgloss.Top,
		titleStyle.Width(m.ui.width).Render("Resttest"),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			panel0,
			lipgloss.JoinVertical(lipgloss.Top, panel1, panel2)),
	)
	return render
}

func main() {
	f, _ := os.Create("debug.log")
	log.SetOutput(f)
	defer f.Close()

	user_endpoints, err := loadAPIConfig("./user/endpoints.yaml")
	if err != nil {
		fmt.Errorf("error trying to read the api config, %w", err)
	}

	first_endpoint := user_endpoints.Endpoints[0]
	full_url := constructRequest(first_endpoint)
	fmt.Println(full_url)

	// Then in your code
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("there was an error!")
		fmt.Printf("%s", err)
		os.Exit(1)
	}

}
