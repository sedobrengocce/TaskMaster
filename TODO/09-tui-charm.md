# 09 - TUI con Bubbletea / Bubbles / Lip Gloss

## Descrizione
Creare un'interfaccia terminale interattiva (TUI) usando il framework Charm (Bubbletea per il modello, Bubbles per i componenti, Lip Gloss per lo styling). La TUI sara un comando della CLI (`taskmaster tui`) e offrira navigazione tra progetti, task e vista settimanale.

## Sotto-task

### Setup
- [ ] Aggiungere dipendenze Charm:
  - `github.com/charmbracelet/bubbletea`
  - `github.com/charmbracelet/bubbles`
  - `github.com/charmbracelet/lipgloss`
- [ ] Creare `cmd/cli/tui/` package

### Modello principale
- [ ] Creare `tui/app.go` — modello root con navigazione tra viste
- [ ] Implementare state machine: Login -> ProjectList -> TaskList -> WeeklyView
- [ ] Gestire keybinding globali: `q`/`Ctrl+C` per uscire, `?` per help, `Esc` per tornare indietro

### Vista Login
- [ ] Form di login con campi email e password (bubbles `textinput`)
- [ ] Feedback visivo per errori di autenticazione
- [ ] Skip se token gia presente e valido

### Vista Lista Progetti
- [ ] Lista progetti con bubbles `list`
- [ ] Indicatore visivo per progetti condivisi
- [ ] Azioni: `Enter` per aprire, `n` per nuovo, `d` per eliminare, `s` per condividere
- [ ] Colore del progetto visualizzato con Lip Gloss

### Vista Lista Task
- [ ] Lista task del progetto selezionato con bubbles `list`
- [ ] Checkbox per stato di completamento
- [ ] Azioni: `Space` per toggle completamento, `n` per nuovo, `d` per eliminare
- [ ] Indicatore priorita con colori (Lip Gloss)
- [ ] Filtro per tipo (single/repetitive) con `tab`

### Vista Settimanale
- [ ] Tabella settimanale (lun-dom) con bubbles `table`
- [ ] Righe = task, Colonne = giorni della settimana
- [ ] Celle con checkmark per i giorni completati
- [ ] Navigazione tra settimane con frecce sinistra/destra
- [ ] Header con date della settimana corrente
- [ ] Styling con Lip Gloss: bordi, colori, allineamento

### Componenti condivisi
- [ ] Barra di stato in basso con info utente e navigazione
- [ ] Spinner per operazioni di rete
- [ ] Dialog di conferma per eliminazioni
- [ ] Notifiche toast per successo/errore

### Integrazione
- [ ] Registrare comando `taskmaster tui` in Cobra
- [ ] Riutilizzare il client HTTP dalla CLI

## File coinvolti

- `cmd/cli/tui/` — nuova directory
- `cmd/cli/tui/app.go` — modello root
- `cmd/cli/tui/login.go` — vista login
- `cmd/cli/tui/projects.go` — vista progetti
- `cmd/cli/tui/tasks.go` — vista task
- `cmd/cli/tui/weekly.go` — vista settimanale
- `cmd/cli/tui/styles.go` — tema e stili Lip Gloss
- `cmd/cli/tui/components.go` — componenti riutilizzabili
- `cmd/cli/commands/tui.go` — comando Cobra
- `go.mod` — nuove dipendenze

## Dipendenze

- `08-cli-cobra-viper.md` — la TUI e un comando della CLI
- `06-weekly-view.md` — serve l'API per la vista settimanale
