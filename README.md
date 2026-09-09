# Mana

Mana is a small, responsive conference catering guide. It supports multiple events, each with its own menu URL, scheduled meals, food trucks, coffee, drinks, and snacks. German and English are fully supported.

## Features

- responsive and accessible design without a frontend framework
- server-rendered HTML
- DE/EN switch that follows the browser's language preferences and falls back to German
- separate event URLs mapped to menu files in `content/events.yaml`
- YAML as the single source of truth for menu content
- templates and assets embedded in one Go binary
- gzip compression and long-lived browser caching for slow or crowded Wi-Fi
- health endpoint and secure HTTP headers
- multi-stage Docker image running as a non-root user
- hardened Docker Compose configuration with a read-only container filesystem

## Start with Docker Compose

```bash
docker compose up --build
```

The example event is available at [http://localhost:8080/example-conference](http://localhost:8080/example-conference). The root page only displays a QR-code instruction. The health endpoint is exposed at `/healthz`.

## Edit the menu

Each event has its own menu file, selected by `content/events.yaml`. The example conference uses `content/menu.yaml`, and the community event uses `content/community-day.yaml`. Names and descriptions have German and English variants:

```yaml
name:
  de: Gemüse-Curry
  en: Vegetable curry
```

Docker Compose bind-mounts the `content` directory from the host. Edit the relevant event's menu file with any host-side editor. Mana checks for updates at most twice per second, so no container restart or rebuild is required. If an edit temporarily produces invalid YAML, Mana keeps serving the most recent valid menu.

The directory mount is read-only (`:ro`) from the container's perspective. This prevents the application from accidentally changing the source file while it remains fully editable on the host, including with editors that save by replacing the file. Reload the page in the browser to see an update.

Without `CONTENT_DIR`, the application uses the event manifest and menus embedded at build time.

## Local development

Go 1.24 or newer is required.

```bash
go mod download
go run .
```

Run tests and create a local binary:

```bash
go test ./...
go build -o bin/mana .
```

Run `make test` to include the JavaScript tests (requires Node.js with `node --test` support).

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port used by the web server |
| `CONTENT_DIR` | empty | Optional directory containing events.yaml and event menus |

## Optional prices

Add `price` to a service (a whole meal), an individual meal item, or a permanent coffee/drink/snack in the event's menu file:

```yaml
# Within days[].services[]:
- id: lunch
  price: 12.50
  # Keep the existing title, subtitle, from and until fields.
  items:
    - id: vegetable_curry
      price: 8.50
      name: {de: Gemüse-Curry, en: Vegetable curry}
```

Prices are euro amounts with a decimal point and at most two decimal places. Negative amounts and invalid values are rejected. Omit `price` or set it to `null` to hide it; `price: 0` explicitly displays zero. Meal and item prices are independent and are not added together or inherited.

The language switch displays German (`12,50 €`) or English (`€12.50`) formatting in the featured meal, full schedule, drinks, and snacks. External menu prices reload just like other menu content.

Coffee is configured separately under `permanent.coffee`, other drinks under `permanent.drinks`, and snacks under `permanent.snacks`. Coffee items support the same optional prices and translations. The Coffee section is hidden when its list is empty.

For optional size prices, replace `price` with either or both size fields on a meal or item:

```yaml
- id: cappuccino
  name: {de: Cappuccino, en: Cappuccino}
  price_normal: 3.20
  price_large: 4.20
```

Size labels are Normal/Groß in German and Regular/Large in English. Missing or null sizes are hidden, and zero is displayed. Do not combine a single `price` with size prices on the same meal or item.

## Food trucks

Add an optional `food_trucks` list to each entry in `days`, alongside `services`:

```yaml
- date: "2026-10-12"
  food_trucks:
    - id: pita_stop
      name: {de: Pita-Pause, en: The Pita Stop}
      description: {de: Frische Pita und Falafel., en: Fresh pita and falafel.}
      location: {de: Innenhof, en: Courtyard}
      from: "11:30"
      until: "15:00"
  services:
    # Existing conference meals go here.
```

Add as many trucks as needed per day, using unique IDs within that day. Names, descriptions, and locations require both German and English. Times use `HH:MM`, with `until` later than `from` on the same day. Trucks appear in a separate section for the selected day. Omit `food_trucks` or use `food_trucks: []` to hide that day's section.

## Sold out

Add `sold_out: true` to any service (whole meal) or item, including coffee, drinks, and snacks:

```yaml
- id: cappuccino
  name: {de: Cappuccino, en: Cappuccino}
  price: 3.20
  sold_out: true
```

The item stays visible with an **Ausverkauft / Sold out** badge in place of its prices. Set `sold_out: false` or remove the field to restore normal display. A meal flag labels the whole service; individual item flags are independent. With `CONTENT_DIR` enabled, changes reload with the menu.

## Development with automatic reload

Run `make dev` to rebuild and restart when Go, HTML, CSS, JavaScript, or YAML changes. This uses Air from the separate `dev.mod` / `dev.sum` module files (Go 1.26+), while the application module remains on Go 1.24. Refresh the browser to see changes.

Air is development-only: Docker excludes its module files and local build outputs, builds only the application, and copies only the Mana binary into the final runtime image. The container starts Mana directly.

## Automatic day and featured meal

`conference.timezone` sets the conference clock (default: `Europe/Berlin`). The page selects today, the next configured day if today has no menu, or the final day after the conference. The featured meal is the currently running service, then the next upcoming service, or the final service once the day ends. Future days show their first meal; past days show their last. Overlapping services feature the one that started most recently. Sold-out services remain visible with their badge.

The clock updates every 15 seconds and when returning to the tab. Selecting a day or opening a topic link keeps that day selected until reload; its featured meal still updates. Without JavaScript, the first day and first meal remain the fallback.

## Multiple events

Map event paths to menu files in `content/events.yaml`:

```yaml
events:
  - path: /example-conference
    menu: menu.yaml
  - path: /community-day
    menu: community-day.yaml
```

Each menu uses the same conference, days, food trucks, and refreshments format. Paths are unique lowercase slugs; `/`, `/healthz`, and `/static` are reserved. Generate each venue QR code for its full event URL. `/` displays only a centered German/English instruction to scan the venue QR code, and unknown paths return 404.

Event URLs are intentionally public and require no login or access token. Anyone who knows, guesses, or receives an event URL can open its menu directly. The QR code provides a convenient link; scanning it is not required for access. The root page does not list events.

`make dev` and Docker Compose use `CONTENT_DIR` to read external files. Menu edits reload independently, retaining each event's last valid menu on invalid edits. Restart the app after changing `events.yaml` to add, remove, or rename event paths. `MENU_PATH` has been replaced by `CONTENT_DIR`.
