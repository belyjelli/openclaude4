package providerwizard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWizard_OpenAI_BackAndResult(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	w := New()
	if w.StepKind() != StepMenu {
		t.Fatalf("want menu, got %v", w.StepKind())
	}
	if err := w.SelectMenuIndex(0); err != nil { // openai
		t.Fatal(err)
	}
	if w.step != stOpenAIBaseMenu {
		t.Fatalf("want openai base menu, got %d", w.step)
	}
	customIdx := len(w.MenuOptions()) - 1
	if err := w.SelectMenuIndex(customIdx); err != nil {
		t.Fatal(err)
	}
	if w.step != stOpenAIBaseText {
		t.Fatalf("want base text step, got %d", w.step)
	}
	if err := w.SubmitText("https://api.example/v1"); err != nil {
		t.Fatal(err)
	}
	if w.step != stOpenAIModelText {
		t.Fatalf("want model text, got %d", w.step)
	}
	if !w.Back() {
		t.Fatal("expected back to base URL step")
	}
	if w.step != stOpenAIBaseText {
		t.Fatalf("after back want base text, got %d", w.step)
	}
	if err := w.SubmitText("https://api.other/v1"); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitText("gpt-other"); err != nil {
		t.Fatal(err)
	}
	if !w.Finished() || w.Cancelled() {
		t.Fatal("want success finish")
	}
	got := w.Result()
	if !strings.Contains(got, `name: openai`) || !strings.Contains(got, `gpt-other`) {
		t.Fatalf("result missing expected yaml: %q", got)
	}
	if !strings.Contains(got, `https://api.other/v1`) {
		t.Fatalf("result missing base_url: %q", got)
	}
}

func TestWizard_OpenAI_EmptyBase(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	w := New()
	_ = w.SelectMenuIndex(0)
	_ = w.SelectMenuIndex(0) // default official base
	_ = w.SubmitText("m1")
	if !strings.Contains(w.Result(), `model: "m1"`) {
		t.Fatal(w.Result())
	}
	if strings.Contains(w.Result(), "base_url") {
		t.Fatal("should omit base_url")
	}
}

func TestWizard_OpenAI_EnvBaseURLMenu(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "https://proxy.example/v1")
	w := New()
	_ = w.SelectMenuIndex(0)
	if w.openaiBaseEnvIdx < 0 {
		t.Fatal("expected env row in base menu")
	}
	if err := w.SelectMenuIndex(w.openaiBaseEnvIdx); err != nil {
		t.Fatal(err)
	}
	if w.openaiBase != "https://proxy.example/v1" {
		t.Fatalf("base: %q", w.openaiBase)
	}
}

func TestWizard_CancelAtRoot(t *testing.T) {
	w := New()
	w.Cancel()
	if !w.Cancelled() || !w.Finished() {
		t.Fatal()
	}
}

func TestWizard_ParseBackAtRoot(t *testing.T) {
	w := New()
	if w.Back() {
		t.Fatal("back at root should fail")
	}
}

func TestListOllamaModelTags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2"},{"name":"mistral"}]}`))
	}))
	defer srv.Close()
	tags, err := ListOllamaModelTags(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 || tags[0] != "llama3.2" {
		t.Fatalf("%v", tags)
	}
}

func TestWizard_Ollama_HostFetch_MenuFinish(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2"}]}`))
	}))
	defer srv.Close()

	w := New()
	if err := w.SelectMenuIndex(1); err != nil { // ollama
		t.Fatal(err)
	}
	if err := w.SubmitText(srv.URL); err != nil {
		t.Fatal(err)
	}
	if w.step != stOllamaModelMenu {
		t.Fatalf("want model menu, got step %d", w.step)
	}
	if err := w.SelectMenuIndex(0); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(w.Result(), "llama3.2") || !strings.Contains(w.Result(), "ollama") {
		t.Fatal(w.Result())
	}
}

func TestNormalizeOpenRouterBase(t *testing.T) {
	if s := normalizeOpenRouterBase(""); s != "" {
		t.Fatalf("%q", s)
	}
	if s := normalizeOpenRouterBase("https://openrouter.ai/api/v1"); s != "" {
		t.Fatalf("want empty for default host, got %q", s)
	}
	if s := normalizeOpenRouterBase("https://custom.example/v1"); s != "https://custom.example/v1" {
		t.Fatalf("got %q", s)
	}
}
