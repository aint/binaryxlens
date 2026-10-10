(function () {
	const I18N = {
		en: {
			"lang.label": "Language",
			"index.lede": "Blockchain data analysis for tokenized properties on",
			"index.reportsUpdated": "Reports updated",
			"index.reportProblem": "Report a problem",

			"report.title": "Project report",
			"report.invalidJson": "Invalid JSON in #project-data.",
			"aria.summary": "Project summary",
			"aria.charts": "Property charts",
			"aria.holders": "Project holders",

			"summary.properties": "Properties",
			"summary.totalSupply": "Total supply",
			"summary.bought": "Bought (cumulative)",
			"summary.tokens": { one: "{n} token", other: "{n} tokens" },

			"type.construction": "construction",
			"type.rental": "rental",
			"type.redeemed": "redeemed",
			"type.unknown": "unknown",

			"rental.onTime": "on time",
			"rental.delay": { one: "+{n} quarter", other: "+{n} quarters" },
			"rental.notStarted": "not started",
			"rental.dates": "Expected rental start: {expected} · Actual start: {actual}",
			"rental.years": { one: "{n} year", other: "{n} years" },

			"price.area": "Area, m²",
			"price.age": "Age",
			"price.binaryx": "Price, Binaryx",
			"price.market": "Market, USD/m²",
			"price.valueTier": "Estimated value",
			"price.aiNote": "AI analysis, {when}. Market value and estimated value may be inaccurate.",
			"value.fair": "Fair",
			"value.rich": "Rich",
			"value.overpriced": "Overpriced",
			"value.underpriced": "Underpriced",

			"initialSales": "Initial sales",
			"noSeries": "No series",
			"noProperties": "No properties loaded",
			"resetZoom": "Reset zoom",
			"chart.weeklyVolume": "Weekly volume",
			"chart.cumulative": "Cumulative",

			"eta.title": "ETA",
			"eta.none": "No ETA rows.",
			"eta.band": "Band",
			"eta.window": "Window",
			"eta.rate": "Avg Δ",
			"eta.days": "Days",
			"eta.date": "Date",
			"eta.last1": "last 1 week",
			"eta.last4": "last 4 weeks",
			"eta.all": "full history (all weeks)",

			"secondary.title": "Weekly secondary volume",
			"secondary.market": "Secondary market",
			"secondary.noTransfers": "no wallet-to-wallet transfers yet",
			"secondary.txs": "Secondary txs",

			"holders.title": "Holders",
			"holders.top": "Top {n} holders",
			"holders.all": "all",
			"holders.allTitle": "All holders",
			"holders.limit": "Number of holders",
			"holders.none": "No holders.",
			"holders.global": "Top global holders",
			"holders.showing": "Showing {shown} of {total} holders (by total balance).",
			"holders.exited": "Exited",
			"col.address": "Address",
			"col.propertyNames": "Property names",
			"col.balance": "Tokens / USDT, total",
			"col.p2pBought": "Tokens / USDT, P2P",
			"col.initialBought": "Tokens / USDT, initial sale",
			"col.week": "Week Δ",
			"col.month": "Month Δ",
			"col.supplyPct": "% supply",
			"col.tier": "Tier",

			"tier.distribution": "Tier distribution",
			"tier.holdersChart": "Holders by tier",
			"tier.supplyChart": "Supply by tier",
			"tier.noData": "No tier data",
			"tier.tokens": "Tokens: {v}",
			"tier.supplyPct": "% supply: {v}%",
			"tier.holdersPct": "% holders: {v}%",
			"tier.shrimp": "🦐 Shrimp",
			"tier.crab": "🦀 Crab",
			"tier.fish": "🐟 Fish",
			"tier.dolphin": "🐬 Dolphin",
			"tier.shark": "🦈 Shark",
			"tier.whale": "🐋 Whale"
		},
		uk: {
			"lang.label": "Мова",
			"index.lede": "Аналіз блокчейн-даних токенізованої нерухомості на",
			"index.reportsUpdated": "Звіти оновлено",
			"index.reportProblem": "Повідомити про проблему",

			"report.title": "Звіт по проєкту",
			"report.invalidJson": "Некоректний JSON у #project-data.",
			"aria.summary": "Підсумок проєкту",
			"aria.charts": "Графіки об'єктів",
			"aria.holders": "Власники проєкту",

			"summary.properties": "Об'єкти",
			"summary.totalSupply": "Загальна емісія",
			"summary.bought": "Куплено (усього)",
			"summary.tokens": { one: "{n} токен", few: "{n} токени", many: "{n} токенів", other: "{n} токена" },

			"type.construction": "будівництво",
			"type.rental": "оренда",
			"type.redeemed": "викуплено",
			"type.unknown": "невідомо",

			"rental.onTime": "вчасно",
			"rental.delay": { one: "+{n} квартал", few: "+{n} квартали", many: "+{n} кварталів", other: "+{n} кварталу" },
			"rental.notStarted": "не розпочато",
			"rental.dates": "Очікуваний старт оренди: {expected} · Фактичний старт: {actual}",
			"rental.years": { one: "{n} рік", few: "{n} роки", many: "{n} років", other: "{n} року" },

			"price.area": "Площа, м²",
			"price.age": "Вік",
			"price.binaryx": "Ціна, Binaryx",
			"price.market": "Ринок, USD/м²",
			"price.valueTier": "Оцінка вартості",
			"price.aiNote": "Аналіз ШІ, {when}. Ринкова вартість і оцінка вартості можуть бути неточними.",
			"value.fair": "Справедлива",
			"value.rich": "Висока",
			"value.overpriced": "Завищена",
			"value.underpriced": "Занижена",

			"initialSales": "Первинні продажі",
			"noSeries": "Немає даних",
			"noProperties": "Об'єкти не завантажено",
			"resetZoom": "Скинути масштаб",
			"chart.weeklyVolume": "Тижневий обсяг",
			"chart.cumulative": "Накопичено",

			"eta.title": "Прогноз",
			"eta.none": "Немає даних прогнозу.",
			"eta.band": "Діапазон",
			"eta.window": "Період",
			"eta.rate": "Сер. Δ",
			"eta.days": "Днів",
			"eta.date": "Дата",
			"eta.last1": "останній 1 тиждень",
			"eta.last4": "останні 4 тижні",
			"eta.all": "уся історія (усі тижні)",

			"secondary.title": "Тижневий обсяг вторинного ринку",
			"secondary.market": "Вторинний ринок",
			"secondary.noTransfers": "переказів між гаманцями ще немає",
			"secondary.txs": "Вторинних транзакцій",

			"holders.title": "Власники",
			"holders.top": "Топ-{n} власників",
			"holders.all": "усі",
			"holders.allTitle": "Усі власники",
			"holders.limit": "Кількість власників",
			"holders.none": "Немає власників.",
			"holders.global": "Топ власників проєкту",
			"holders.showing": "Показано {shown} з {total} власників (за загальним балансом).",
			"holders.exited": "Вийшли",
			"col.address": "Адреса",
			"col.propertyNames": "Об'єкти",
			"col.balance": "Токени / USDT, всього",
			"col.p2pBought": "Токени / USDT, P2P",
			"col.initialBought": "Токени / USDT, первинний продаж",
			"col.week": "Тиждень Δ",
			"col.month": "Місяць Δ",
			"col.supplyPct": "% емісії",
			"col.tier": "Рівень",

			"tier.distribution": "Розподіл за рівнями",
			"tier.holdersChart": "Власники за рівнями",
			"tier.supplyChart": "Емісія за рівнями",
			"tier.noData": "Немає даних за рівнями",
			"tier.tokens": "Токени: {v}",
			"tier.supplyPct": "% емісії: {v}%",
			"tier.holdersPct": "% власників: {v}%",
			"tier.shrimp": "🦐 Креветка",
			"tier.crab": "🦀 Краб",
			"tier.fish": "🐟 Риба",
			"tier.dolphin": "🐬 Дельфін",
			"tier.shark": "🦈 Акула",
			"tier.whale": "🐋 Кит"
		}
	};
	const STORAGE_KEY = "lang";

	function storedLang() {
		try {
			return localStorage.getItem(STORAGE_KEY);
		} catch (e) {
			return null;
		}
	}

	// ?lang= wins over the saved choice, which wins over the browser language.
	function detectLang() {
		const candidates = [
			new URLSearchParams(location.search).get("lang"),
			storedLang(),
			(navigator.language || "").slice(0, 2).toLowerCase()
		];
		for (const code of candidates) {
			if (code && I18N[code]) return code;
		}
		return "en";
	}

	const lang = detectLang();
	const pluralRules = new Intl.PluralRules(lang);
	document.documentElement.lang = lang;

	function has(key) {
		return key in I18N[lang] || key in I18N.en;
	}

	// Plural entries are objects keyed by Intl.PluralRules category and pick a form by vars.n.
	function t(key, vars) {
		let msg = key in I18N[lang] ? I18N[lang][key] : I18N.en[key];
		if (msg == null) return key;
		if (typeof msg === "object") {
			msg = msg[pluralRules.select(Number(vars && vars.n))] || msg.other;
		}
		if (!vars) return msg;
		return msg.replace(/\{(\w+)\}/g, function (match, name) {
			return vars[name] != null ? String(vars[name]) : match;
		});
	}

	function applyStatic() {
		document.querySelectorAll("[data-i18n]").forEach(function (el) {
			el.textContent = t(el.dataset.i18n);
		});
		document.querySelectorAll("[data-i18n-aria]").forEach(function (el) {
			el.setAttribute("aria-label", t(el.dataset.i18nAria));
		});
	}

	function switchTo(code) {
		try {
			localStorage.setItem(STORAGE_KEY, code);
		} catch (e) {
			// The ?lang= param below still carries the choice for this page.
		}
		const url = new URL(location.href);
		url.searchParams.set("lang", code);
		location.href = url.toString();
	}

	function mountSwitcher() {
		const slot = document.getElementById("lang-switch");
		if (!slot) return;
		slot.setAttribute("role", "group");
		slot.setAttribute("aria-label", t("lang.label"));
		Object.keys(I18N).forEach(function (code) {
			const button = document.createElement("button");
			button.type = "button";
			button.textContent = code.toUpperCase();
			button.setAttribute("aria-pressed", String(code === lang));
			button.addEventListener("click", function () {
				if (code !== lang) switchTo(code);
			});
			slot.appendChild(button);
		});
	}

	applyStatic();
	mountSwitcher();
	window.i18n = { lang: lang, t: t, has: has };
})();
