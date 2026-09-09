(() => {
  const root = document.documentElement;
  const languageButtons = [...document.querySelectorAll("[data-language]")];
  const languageContent = [...document.querySelectorAll("[data-lang-content]")];
  const dayButtons = [...document.querySelectorAll("[data-day-button]")];
  const dayPanels = [...document.querySelectorAll("[data-day-panel]")];

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
    const today = new Date();
    const todayKey = [today.getFullYear(), String(today.getMonth() + 1).padStart(2, "0"), String(today.getDate()).padStart(2, "0")].join("-");
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
  dayButtons.forEach((button) => button.addEventListener("click", () => selectDay(button.dataset.dayButton)));

  function restoreTopic() {
    const target = document.getElementById(location.hash.slice(1));
    const panel = target?.closest("[data-day-panel]");
    if (!panel) return;
    selectDay(panel.dataset.dayPanel);
    target.scrollIntoView();
  }

  window.addEventListener("hashchange", restoreTopic);
  setLanguage(readLanguage());
  restoreTopic();
})();
