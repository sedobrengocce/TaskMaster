# 10 - Frontend Web con Templ + HTMX

## Descrizione
Creare un frontend web server-side usando Templ (template Go type-safe) e HTMX per interattivita senza JavaScript framework. Il server servira pagine HTML con componenti Templ e HTMX gestira le interazioni dinamiche (completamento task, navigazione, ecc.).

## Sotto-task

### Setup
- [ ] Aggiungere dipendenze: `go get github.com/a-h/templ`
- [ ] Installare `templ` CLI: `go install github.com/a-h/templ/cmd/templ@latest`
- [ ] Creare struttura directory:
  - `web/templates/` — componenti Templ
  - `web/static/` — CSS, asset statici
  - `web/handlers/` — handler specifici per le pagine web
- [ ] Configurare Echo per servire file statici e template

### Layout e navigazione
- [ ] `templates/layout.templ` — layout base HTML con head, nav, footer
- [ ] `templates/nav.templ` — barra di navigazione (progetti, settimanale, profilo)
- [ ] Includere HTMX via CDN o file locale
- [ ] CSS minimale (sistema di design semplice, no framework pesanti)

### Pagine di autenticazione
- [ ] `templates/login.templ` — form di login
- [ ] `templates/register.templ` — form di registrazione
- [ ] Handler per login/register che settano cookie di sessione
- [ ] Redirect post-login alla dashboard

### Dashboard / Vista Progetti
- [ ] `templates/projects.templ` — lista progetti con colori
- [ ] `templates/project_card.templ` — card singolo progetto
- [ ] HTMX: creare progetto inline con `hx-post`
- [ ] HTMX: eliminare progetto con conferma `hx-delete` + `hx-confirm`

### Vista Task per Progetto
- [ ] `templates/tasks.templ` — lista task del progetto
- [ ] `templates/task_item.templ` — singolo task con checkbox
- [ ] HTMX: toggle completamento con `hx-post` su checkbox
- [ ] HTMX: creare task inline
- [ ] HTMX: eliminare task con `hx-delete`
- [ ] Filtri per tipo e priorita

### Vista Settimanale
- [ ] `templates/weekly.templ` — griglia settimanale
- [ ] Tabella con giorni come colonne e task come righe
- [ ] HTMX: navigare tra settimane con `hx-get` e swap del contenuto
- [ ] HTMX: toggle completamento giornaliero con `hx-post`
- [ ] Indicatori visivi per completamento

### Sharing
- [ ] `templates/share_modal.templ` — modale per condividere progetto/task
- [ ] Ricerca utente per email con `hx-get` e debounce
- [ ] Lista utenti con cui e condiviso con opzione di rimozione

### Infrastruttura
- [ ] Aggiungere target `templ generate` nel Makefile
- [ ] Configurare live reload per sviluppo (`templ generate --watch`)
- [ ] Middleware per autenticazione sulle route web (cookie-based)
- [ ] Gestire CSRF token nei form HTMX
- [ ] Aggiungere route group `/web/` in `routes.go` per le pagine

## File coinvolti

- `web/` — nuova directory (tutto)
- `web/templates/*.templ` — componenti Templ
- `web/static/` — CSS e asset
- `web/handlers/` — handler pagine web
- `internal/server/routes.go` — route group web
- `internal/server/server.go` — configurazione file statici
- `Makefile` — target templ generate
- `go.mod` — nuove dipendenze

## Dipendenze

- `01-task-crud-handlers.md` — servono le API
- `02-task-completion-logging.md` — completamento task
- `05-authorization-checks.md` — autorizzazione
- `06-weekly-view.md` — vista settimanale
- `07-cors-hardening.md` — CORS configurato per il dominio web
