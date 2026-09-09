(() => {
  const root = document.documentElement;
  const languageButtons = [...document.querySelectorAll("[data-language]")];
  const languageContent = [...document.querySelectorAll("[data-lang-content]")];
  const dayButtons = [...document.querySelectorAll("[data-day-button]")];
  const dayPanels = [...document.querySelectorAll("[data-day-panel]")];

  let manualDay = false;
  const clock = new Intl.DateTimeFormat("en-GB", {
    timeZone: document.body.dataset.timezone,
    year: "numeric", month: "2-digit", day: "2-digit",
    hour: "2-digit", minute: "2-digit", hourCycle: "h23",
  });

  function conferenceNow() {
    const parts = Object.fromEntries(clock.formatToParts(new Date()).map(({type, value}) => [type, value]));
    return {date: `${parts.year}-${parts.month}-${parts.day}`, time: `${parts.hour}:${parts.minute}`};
  }

  const translations = {
    de: {
      languageLabel: "Sprache wählen",
      dayLabel: "Konferenztage",
      topicLabel: "Direkt zu",
      locale: "de-DE",
      today: "Heute",
    },
    en: {
      languageLabel: "Choose language",
      dayLabel: "Conference days",
      topicLabel: "Jump to",
      locale: "en-GB",
      today: "Today",
    },
  };

  function readLanguage() {
    const preferredLanguages = navigator.languages?.length
      ? navigator.languages
      : [navigator.language];

    for (const language of preferredLanguages) {
      const primaryLanguage = language?.toLowerCase().split("-")[0];
      if (primaryLanguage === "de" || primaryLanguage === "en") {
        return primaryLanguage;
      }
    }

    return "de";
  }

  function formatDates(language) {
    const copy = translations[language];
    const todayKey = conferenceNow().date;
    const formatter = new Intl.DateTimeFormat(copy.locale, { weekday: "short", day: "numeric", month: "short" });

    dayButtons.forEach((button) => {
      const dateKey = button.dataset.date;
      const date = new Date(`${dateKey}T12:00:00`);
      const label = formatter.format(date).replace(/\.$/, "");
      button.querySelector("time").textContent = dateKey === todayKey ? `${copy.today} · ${label}` : label;
    });
  }

  function setLanguage(language) {
    const copy = translations[language];
    root.lang = language;
    languageContent.forEach((element) => {
      element.hidden = element.dataset.langContent !== language;
    });
    languageButtons.forEach((button) => {
      button.setAttribute("aria-pressed", String(button.dataset.language === language));
    });
    document.querySelector(".language-switch").setAttribute("aria-label", copy.languageLabel);
    document.querySelector(".day-switcher").setAttribute("aria-label", copy.dayLabel);
    document.querySelectorAll(".topic-nav").forEach((nav) => nav.setAttribute("aria-label", copy.topicLabel));
    formatDates(language);
  }

  function selectDay(index) {
    dayButtons.forEach((button) => {
      button.setAttribute("aria-pressed", String(button.dataset.dayButton === index));
    });
    dayPanels.forEach((panel) => {
      const active = panel.dataset.dayPanel === index;
      panel.hidden = !active;
      panel.classList.toggle("is-active", active);
    });
  }

  languageButtons.forEach((button) => button.addEventListener("click", () => setLanguage(button.dataset.language)));
  dayButtons.forEach((button) => button.addEventListener("click", () => { manualDay = true; selectDay(button.dataset.dayButton); }));

  function restoreTopic() {
    const target = document.getElementById(location.hash.slice(1));
    const panel = target?.closest("[data-day-panel]");
    if (!panel) return;
    manualDay = true;
    selectDay(panel.dataset.dayPanel);
    target.scrollIntoView();
  }

  function updateSchedule() {
    const now = conferenceNow();
    const days = dayButtons.slice().sort((a, b) => a.dataset.date.localeCompare(b.dataset.date));
    const selected = days.find((button) => button.dataset.date >= now.date) || days.at(-1);
    if (!manualDay && selected) selectDay(selected.dataset.dayButton);
    dayPanels.forEach((panel) => {
      const date = dayButtons.find((button) => button.dataset.dayButton === panel.dataset.dayPanel).dataset.date;
      const meals = [...panel.querySelectorAll("[data-featured-from]")].sort((a, b) => a.dataset.featuredFrom.localeCompare(b.dataset.featuredFrom));
      const time = date < now.date ? "24:00" : date > now.date ? "00:00" : now.time;
      const current = meals.filter((meal) => meal.dataset.featuredFrom <= time && time < meal.dataset.featuredUntil).at(-1);
      const featured = current || meals.find((meal) => meal.dataset.featuredFrom > time) || meals.at(-1);
      meals.forEach((meal) => { meal.hidden = meal !== featured; });
    });
    formatDates(root.lang);
  }

  window.addEventListener("hashchange", restoreTopic);
  setLanguage(readLanguage());
  updateSchedule();
  restoreTopic();
  setInterval(updateSchedule, 15000);
  document.addEventListener("visibilitychange", () => { if (!document.hidden) updateSchedule(); });
})();

// Keep only the trigger here; download the animation after the fifth click.
(() => {
  const trigger = document.querySelector('.app-version');
  if (!trigger) return;
  let clicks = 0;
  let loading = false;
  trigger.addEventListener('click', async () => {
    if (loading || ++clicks < 5) return;
    clicks = 0;
    loading = true;
    try {
      const {startManna} = await import('/static/manna.js?v=2');
      startManna();
    } catch {
      // A failed optional download must not affect the menu. Five clicks retry.
    } finally {
      loading = false;
    }
  });
})();
