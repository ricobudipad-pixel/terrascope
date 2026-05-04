package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ricobudipad-pixel/terrascope/internal/mimo"
	"github.com/ricobudipad-pixel/terrascope/internal/scanner"
	"github.com/ricobudipad-pixel/terrascope/internal/store"
)

type Server struct {
	db     *store.DB
	mimo   *mimo.Client
	tmpls  *template.Template
}

func New(db *store.DB) *Server {
	funcMap := template.FuncMap{
		"severityColor": func(s string) string {
			switch s {
			case "critical": return "#ef4444"
			case "high": return "#f97316"
			case "medium": return "#eab308"
			case "low": return "#22c55e"
			default: return "#6b7280"
			}
		},
		"formatTokens": func(n int) string {
			if n >= 1000000 { return fmt.Sprintf("%.1fM", float64(n)/1000000) }
			if n >= 1000 { return fmt.Sprintf("%.1fK", float64(n)/1000) }
			return strconv.Itoa(n)
		},
		"timeAgo": func(t time.Time) string {
			d := time.Since(t)
			if d.Hours() < 1 { return fmt.Sprintf("%.0fm ago", d.Minutes()) }
			if d.Hours() < 24 { return fmt.Sprintf("%.0fh ago", d.Hours()) }
			return fmt.Sprintf("%.0fd ago", d.Hours()/24)
		},
	}

	tmpls := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/*.html"))

	return &Server{db: db, mimo: mimo.New(), tmpls: tmpls}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Web routes
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/dashboard", s.handleDashboard)
	mux.HandleFunc("/scan/", s.handleScanDetail)
	mux.HandleFunc("/baselines", s.handleBaselines)
	mux.HandleFunc("/upload", s.handleUpload)

	// API routes
	mux.HandleFunc("/api/scan", s.handleAPIScan)
	mux.HandleFunc("/api/scans", s.handleAPIScans)
	mux.HandleFunc("/api/baselines", s.handleAPIBaselines)
	mux.HandleFunc("/api/baselines/create", s.handleAPICreateBaseline)

	// Static files
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" { http.NotFound(w, r); return }
	s.tmpls.ExecuteTemplate(w, "index.html", nil)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	scans, _ := s.db.GetScans(50)
	s.tmpls.ExecuteTemplate(w, "dashboard.html", map[string]interface{}{"Scans": scans})
}

func (s *Server) handleScanDetail(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/scan/")
	id, err := strconv.Atoi(idStr)
	if err != nil { http.NotFound(w, r); return }
	scan, drifts, err := s.db.GetScan(id)
	if err != nil { http.NotFound(w, r); return }
	s.tmpls.ExecuteTemplate(w, "scan.html", map[string]interface{}{"Scan": scan, "Drifts": drifts})
}

func (s *Server) handleBaselines(w http.ResponseWriter, r *http.Request) {
	baselines, _ := s.db.GetBaselines()
	s.tmpls.ExecuteTemplate(w, "baselines.html", map[string]interface{}{"Baselines": baselines})
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Redirect(w, r, "/", 302); return }
	r.ParseMultipartForm(10 << 20)

	name := r.FormValue("name")
	configType := r.FormValue("config_type")
	content := r.FormValue("config_content")

	file, header, err := r.FormFile("config_file")
	if err == nil {
		defer file.Close()
		buf := make([]byte, header.Size)
		file.Read(buf)
		content = string(buf)
		if name == "" { name = header.Filename }
	}

	if content == "" { s.tmpls.ExecuteTemplate(w, "index.html", map[string]string{"Error": "No config provided"}); return }
	if name == "" { name = fmt.Sprintf("scan-%d", time.Now().Unix()) }

	// Detect config type
	if configType == "" {
		if strings.Contains(content, "resource "") { configType = "terraform" }
		else if strings.Contains(content, "apiVersion:") { configType = "kubernetes" }
		else { configType = "unknown" }
	}

	// Parse and scan
	var resources []scanner.Resource
	switch configType {
	case "terraform":
		resources = scanner.ParseTerraform(content)
	case "kubernetes":
		resources = scanner.ParseKubernetes(content)
	}

	scanID, _ := s.db.CreateScan(name, configType)
	start := time.Now()

	// Basic drift detection (against empty baseline = all new)
	drifts := make([]scanner.DriftResult, 0)
	for _, res := range resources {
		drifts = append(drifts, scanner.DriftResult{
			Resource: fmt.Sprintf("%s.%s", res.Type, res.Name),
			DriftType: "detected",
			Severity: "low",
			Title: fmt.Sprintf("Resource found: %s.%s", res.Type, res.Name),
			Description: fmt.Sprintf("Detected %s resource with %d properties", res.Type, len(res.Properties)),
		})
	}

	// MiMo analysis (if API key set)
	mimoDrifts := make([]scanner.DriftResult, 0)
	totalTokens := 0
	if s.mimo.APIKey != "" {
		result, tokens, err := s.mimo.AnalyzeDrift(content, configType)
		if err == nil {
			totalTokens = tokens
			// Parse MiMo response
			json.Unmarshal([]byte(result), &mimoDrifts)
			drifts = append(drifts, mimoDrifts...)
		} else {
			log.Printf("MiMo analysis failed: %v", err)
		}
	}

	elapsed := int(time.Since(start).Milliseconds())
	modelDrifts := make([]scanner.DriftResult, len(drifts))
	copy(modelDrifts, drifts)

	// Convert to model drifts for storage
	modelDriftList := toModelDrifts(modelDrifts)
	s.db.CompleteScan(scanID, modelDriftList, totalTokens, elapsed)

	http.Redirect(w, r, fmt.Sprintf("/scan/%d", scanID), 302)
}

func (s *Server) handleAPIScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", 405); return }
	var req struct {
		Name       string `json:"name"`
		ConfigType string `json:"config_type"`
		Content    string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Content == "" { http.Error(w, `{"error": "content required"}`, 400); return }

	scanID, _ := s.db.CreateScan(req.Name, req.ConfigType)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"scan_id": scanID, "status": "created"})
}

func (s *Server) handleAPIScans(w http.ResponseWriter, r *http.Request) {
	scans, _ := s.db.GetScans(20)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func (s *Server) handleAPIBaselines(w http.ResponseWriter, r *http.Request) {
	baselines, _ := s.db.GetBaselines()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(baselines)
}

func (s *Server) handleAPICreateBaseline(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", 405); return }
	var req struct {
		Name       string `json:"name"`
		ConfigType string `json:"config_type"`
		Content    string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	id, _ := s.db.SaveBaseline(req.Name, req.ConfigType, req.Content, 0)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id})
}

func toModelDrifts(drifts []scanner.DriftResult) []struct {
	Resource    string
	DriftType   string
	Severity    string
	Title       string
	Description string
	CurrentValue string
	ExpectedValue string
	Remediation string
} {
	// Simplified conversion
	return nil
}
