"use strict";

// Window titles come from other programs and only ever land in textContent: parsed as markup,
// a title could run as code on a page that holds the key.

const sliders = [
	{ key: "opacity", min: 10, max: 100 },
	{ key: "length", min: 1, max: 64 },
	{ key: "thickness", min: 1, max: 16 },
	{ key: "gap", min: 0, max: 32 },
	{ key: "radius", min: 2, max: 64 },
	{ key: "dot", min: 1, max: 16 },
	{ key: "outline", min: 0, max: 4 },
];

const uses = {
	"dot": ["dot"],
	"cross": ["length", "thickness", "gap"],
	"cross-dot": ["length", "thickness", "gap", "dot"],
	"circle": ["radius", "thickness"],
	"circle-dot": ["radius", "thickness", "dot"],
	"t": ["length", "thickness", "gap"],
};

const always = ["opacity", "outline"];

const modifiers = [["ctrlKey", "Ctrl"], ["shiftKey", "Shift"], ["altKey", "Alt"], ["metaKey", "Win"]];

const namedKeys = {
	ArrowLeft: "Left", ArrowRight: "Right", ArrowUp: "Up", ArrowDown: "Down",
	Home: "Home", End: "End", PageUp: "PageUp", PageDown: "PageDown",
	Insert: "Insert", Delete: "Delete", Space: "Space",
};

let view = null;
let selected = "";
let mode = "ready";

function byId(id)
{
	return document.getElementById(id);
}

function t(key)
{
	return view && key in view.text ? view.text[key] : key;
}

function same(a, b)
{
	return a.toLowerCase() === b.toLowerCase();
}

function element(tag, attributes, ...children)
{
	const node = document.createElement(tag);
	for (const [name, value] of Object.entries(attributes))
	{
		if (name === "text")
		{
			node.textContent = value;
			continue;
		}
		node.setAttribute(name, value);
	}
	node.append(...children);

	return node;
}

function games(exe)
{
	return "api/games/" + encodeURIComponent(exe);
}

async function api(method, path, body)
{
	const options = { method: method, headers: {} };
	if (body instanceof Blob)
	{
		options.body = body;
		options.headers["Content-Type"] = "image/png";
	}
	else if (body !== undefined)
	{
		options.body = JSON.stringify(body);
		options.headers["Content-Type"] = "application/json";
	}

	let response;
	try
	{
		response = await fetch(path, options);
	}
	catch
	{
		complain(t("error.offline"));
		return null;
	}

	if (response.status === 403)
	{
		complain(t("error.stale"));
		return null;
	}

	const answer = await response.json().catch(() => null);
	if (!response.ok)
	{
		refuse(answer);
		return null;
	}

	byId("problem").hidden = true;

	return answer;
}

function refuse(answer)
{
	const code = answer && answer.error ? answer.error : "failed";
	if (code === "failed")
	{
		complain(t("error.failed") + " " + (answer && answer.detail ? answer.detail : ""));
		return;
	}

	complain(t("error." + code));
}

function complain(message)
{
	const box = byId("problem");
	box.textContent = message;
	box.hidden = false;
}

// One change on the way at a time, and of the ones that pile up meanwhile only the newest goes:
// a dragged slider would otherwise queue a request for every pixel it passed.
function latest(send)
{
	let running = false;
	let waiting = null;

	return async function (value)
	{
		waiting = value;
		if (running)
		{
			return;
		}

		running = true;
		while (waiting !== null)
		{
			const next = waiting;
			waiting = null;
			await send(next);
		}
		running = false;
	};
}

async function load()
{
	const fresh = await api("GET", "api/state");
	if (fresh)
	{
		show(fresh);
	}
}

async function change(method, path, body)
{
	const fresh = await api(method, path, body);
	if (fresh)
	{
		show(fresh);
	}

	return fresh;
}

function show(fresh)
{
	view = fresh;
	document.documentElement.lang = view.language;
	buildShapes();
	for (const node of document.querySelectorAll("[data-text]"))
	{
		node.textContent = t(node.dataset.text);
	}

	byId("visible").checked = view.settings.visible;
	byId("version").textContent = view.version;
	if (!view.settings.games.some((game) => same(game.exe, selected)))
	{
		selected = view.settings.games.length > 0 ? view.settings.games[0].exe : "";
	}

	showGames();
	showGame();
	showHotkeys();
}

function current()
{
	return view.settings.games.find((game) => same(game.exe, selected));
}

function showGames()
{
	const list = byId("games");
	list.replaceChildren();
	for (const game of view.settings.games)
	{
		const button = element("button", { type: "button", class: same(game.exe, selected) ? "item chosen" : "item" },
			element("span", { class: "label", text: game.name || game.exe }),
			element("span", { class: "exe", text: game.exe }));
		if (!game.enabled)
		{
			button.append(element("span", { class: "badge", text: t("gameOff") }));
		}
		button.addEventListener("click", () => choose(game.exe));
		list.append(element("li", {}, button));
	}

	byId("no-games").hidden = view.settings.games.length > 0;
}

function choose(exe)
{
	selected = exe;
	showGames();
	showGame();
}

function showGame()
{
	const game = current();
	byId("game").hidden = !game;
	if (!game)
	{
		return;
	}

	byId("game-name").value = game.name;
	byId("game-enabled").checked = game.enabled;
	byId("game-exe").textContent = game.exe;
	byId("offset-x").value = game.offset.x;
	byId("offset-y").value = game.offset.y;
	mode = game.crosshair.image ? "picture" : "ready";
	showCrosshair(game.crosshair);
	showSaved();
}

function buildShapes()
{
	const box = byId("shapes");
	if (box.childElementCount > 0)
	{
		return;
	}

	for (const shape of view.shapes)
	{
		const button = element("button", { type: "button", class: "shape", "data-shape": shape, "data-text": "shape." + shape });
		button.addEventListener("click", () => changeCrosshair((c) => withValue(c, "shape", shape)));
		box.append(button);
	}
}

function buildSliders()
{
	const box = byId("sliders");
	for (const slider of sliders)
	{
		const input = element("input", { type: "range", id: "slider-" + slider.key, min: slider.min, max: slider.max, step: 1 });
		const caption = element("span", { class: "caption" }, element("span", { "data-text": slider.key }), element("output", { id: "value-" + slider.key }));
		input.addEventListener("input", () => changeCrosshair((c) => withValue(c, slider.key, Number(input.value))));
		box.append(element("label", { class: "field", id: "field-" + slider.key }, caption, input));
	}
}

function valueOf(c, key)
{
	return key === "outline" ? c.outline.width : c[key];
}

function withValue(c, key, value)
{
	const next = structuredClone(c);
	if (key === "outline")
	{
		next.outline.width = value;
		return next;
	}

	next[key] = value;

	return next;
}

function showCrosshair(c)
{
	byId("ready").hidden = mode !== "ready";
	byId("picture").hidden = mode !== "picture";
	byId("kind-ready").setAttribute("aria-selected", String(mode === "ready"));
	byId("kind-picture").setAttribute("aria-selected", String(mode === "picture"));

	for (const button of byId("shapes").children)
	{
		button.classList.toggle("chosen", button.dataset.shape === c.shape);
	}
	for (const slider of sliders)
	{
		byId("slider-" + slider.key).value = valueOf(c, slider.key);
		byId("value-" + slider.key).textContent = valueOf(c, slider.key);
		byId("field-" + slider.key).hidden = !always.includes(slider.key) && !(uses[c.shape] || []).includes(slider.key);
	}
	byId("color").value = c.color.toLowerCase();
	byId("outline-color").value = c.outline.color.toLowerCase();
	byId("scale").value = c.scale;
	byId("scale-value").textContent = c.scale;

	showPreview(c);
}

function showPreview(c)
{
	const address = "api/preview?crosshair=" + encodeURIComponent(JSON.stringify(c));
	byId("preview-dark").src = address;
	byId("preview-light").src = address;
}

function enlarge(event)
{
	const image = event.target;
	const zoom = Math.max(1, Math.min(4, Math.floor(160 / image.naturalWidth)));
	image.style.width = image.naturalWidth * zoom + "px";
}

// The page answers a change on the spot, and the program gets it shortly after: the reply only
// matters when it is a refusal, and then the state comes back whole from the program.
const sendCrosshair = latest(async function (edit)
{
	if (!await api("PUT", games(edit.exe) + "/crosshair", edit.crosshair))
	{
		await load();
	}
});

function changeCrosshair(edit)
{
	const game = current();
	if (!game)
	{
		return;
	}

	game.crosshair = edit(game.crosshair);
	showCrosshair(game.crosshair);
	sendCrosshair({ exe: game.exe, crosshair: game.crosshair });
}

function showSaved()
{
	const list = byId("saved");
	list.replaceChildren();
	view.settings.saved.forEach(function (entry, index)
	{
		const { name, ...crosshair } = entry;
		const use = element("button", { type: "button", text: t("use") });
		const forget = element("button", { type: "button", class: "quiet", text: t("forget") });
		use.addEventListener("click", function ()
		{
			mode = crosshair.image ? "picture" : "ready";
			changeCrosshair(() => structuredClone(crosshair));
		});
		forget.addEventListener("click", () => change("DELETE", "api/saved/" + index));
		list.append(element("li", { class: "saved-entry" }, element("span", { class: "label", text: name }), use, forget));
	});

	byId("no-saved").hidden = view.settings.saved.length > 0;
}

function showHotkeys()
{
	const box = byId("hotkeys");
	box.replaceChildren();
	for (const [field, text] of Object.entries(view.settings.hotkeys))
	{
		const input = element("input", { type: "text", class: "hotkey", value: text, spellcheck: "false", autocomplete: "off" });
		input.addEventListener("keydown", (event) => capture(event, field, input));
		input.addEventListener("change", () => setHotkey(field, input.value));

		const row = element("div", { class: "hotkey-row" }, element("span", { class: "label", text: t("hotkey." + field) }), input);
		const taken = view.taken.find((entry) => entry.field === field);
		if (taken)
		{
			row.append(warning(field, taken));
		}
		box.append(row);
	}
}

function warning(field, taken)
{
	const note = element("p", { class: "taken" }, element("span", { text: taken.hotkey + " " + t("taken") }));
	if (taken.suggestion)
	{
		const take = element("button", { type: "button", text: t("take") + " " + taken.suggestion });
		take.addEventListener("click", () => setHotkey(field, taken.suggestion));
		note.append(" ", take);
	}

	return note;
}

// With Ctrl, Alt or Win held the field takes the keys pressed; without them it is plain text,
// so a hotkey the browser keeps for itself can still be typed in by hand.
function capture(event, field, input)
{
	const held = modifiers.filter(([flag]) => event[flag]).map(([, name]) => name);
	if (!held.some((name) => name !== "Shift"))
	{
		return;
	}

	const key = namedKeys[event.code] || (/^(?:Key([A-Z])|Digit([0-9])|(F[0-9]{1,2})|(Numpad[0-9]))$/.exec(event.code) || []).slice(1).find(Boolean);
	if (!key)
	{
		return;
	}

	event.preventDefault();
	const arrows = view.arrows.includes(field);
	if (arrows && !["Left", "Right", "Up", "Down"].includes(key))
	{
		return;
	}

	input.value = arrows ? held.join("+") : held.concat(key).join("+");
	setHotkey(field, input.value);
}

async function setHotkey(field, text)
{
	const keys = Object.assign({}, view.settings.hotkeys, { [field]: text });
	if (!await change("PUT", "api/hotkeys", keys))
	{
		showHotkeys();
	}
}

async function startAdding()
{
	byId("adding").hidden = false;
	const choices = await api("GET", "api/windows");
	const list = byId("windows");
	list.replaceChildren();
	for (const choice of choices || [])
	{
		const button = element("button", { type: "button", class: "item" },
			element("span", { class: "label", text: choice.title }),
			element("span", { class: "exe", text: choice.exe }));
		button.addEventListener("click", () => add(choice));
		list.append(element("li", {}, button));
	}

	byId("no-windows").hidden = choices === null || choices.length > 0;
}

async function add(choice)
{
	if (await change("POST", "api/games", { exe: choice.exe, name: choice.title }))
	{
		byId("adding").hidden = true;
		choose(choice.exe);
	}
}

function nudge(event)
{
	const step = Number(byId("step").value);
	const button = event.currentTarget;
	const game = current();
	if (game)
	{
		change("POST", games(game.exe) + "/nudge", { x: step * Number(button.dataset.x), y: step * Number(button.dataset.y) });
	}
}

function wire()
{
	byId("visible").addEventListener("change", (event) => change("PUT", "api/visible", event.target.checked));
	byId("add-game").addEventListener("click", startAdding);
	byId("refresh-windows").addEventListener("click", startAdding);
	byId("cancel-adding").addEventListener("click", function ()
	{
		byId("adding").hidden = true;
	});

	byId("game-name").addEventListener("change", (event) => change("PATCH", games(selected), { name: event.target.value }));
	byId("game-enabled").addEventListener("change", (event) => change("PATCH", games(selected), { enabled: event.target.checked }));
	byId("remove-game").addEventListener("click", function ()
	{
		if (confirm(t("confirm")))
		{
			change("DELETE", games(selected));
		}
	});

	byId("kind-ready").addEventListener("click", function ()
	{
		mode = "ready";
		changeCrosshair((c) => withValue(c, "image", ""));
	});
	byId("kind-picture").addEventListener("click", function ()
	{
		mode = "picture";
		showCrosshair(current().crosshair);
	});
	byId("color").addEventListener("input", (event) => changeCrosshair((c) => withValue(c, "color", event.target.value.toUpperCase())));
	byId("outline-color").addEventListener("input", function (event)
	{
		changeCrosshair(function (c)
		{
			const next = structuredClone(c);
			next.outline.color = event.target.value.toUpperCase();
			return next;
		});
	});
	byId("scale").addEventListener("input", (event) => changeCrosshair((c) => withValue(c, "scale", Number(event.target.value))));
	byId("file").addEventListener("change", upload);
	byId("preview-dark").addEventListener("load", enlarge);
	byId("preview-light").addEventListener("load", enlarge);

	byId("offset-x").addEventListener("change", setOffset);
	byId("offset-y").addEventListener("change", setOffset);
	for (const button of document.querySelectorAll(".pad button"))
	{
		button.addEventListener("click", nudge);
	}
	byId("centre").addEventListener("click", () => change("PUT", games(selected) + "/offset", { x: 0, y: 0 }));

	byId("save").addEventListener("click", save);

	window.addEventListener("focus", load);
}

function setOffset()
{
	change("PUT", games(selected) + "/offset", { x: Number(byId("offset-x").value), y: Number(byId("offset-y").value) });
}

async function upload()
{
	const file = byId("file").files[0];
	byId("file").value = "";
	if (!file)
	{
		return;
	}

	const added = await api("POST", "api/pictures", file);
	if (added)
	{
		mode = "picture";
		changeCrosshair((c) => withValue(c, "image", added.image));
	}
}

async function save()
{
	const game = current();
	const name = byId("save-name").value.trim();
	if (game && name !== "" && await change("POST", "api/saved", { name: name, crosshair: game.crosshair }))
	{
		byId("save-name").value = "";
	}
}

buildSliders();
wire();
load();
