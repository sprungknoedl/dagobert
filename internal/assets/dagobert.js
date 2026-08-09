onload = (event) => {
    // up.log.enable();
    up.log.disable();
    up.network.config.autoCache = (request) => false;
    up.network.config.wrapMethod = false;
    up.layer.config.drawer.position = 'right';
    up.layer.config.drawer.size = 'large';

    // Show a success toast when an overlay is accepted with a {toast: "..."}
    // value, and navigate the root layer when accepted with a {location: "..."}
    // value (both set by the server via the X-Up-Accept-Layer header).
    up.on('up:layer:accepted', function (event) {
        const msg = event.value && event.value.toast;
        if (msg) {
            showToast(msg);
        }
        const location = event.value && event.value.location;
        if (location) {
            up.navigate({ url: location, layer: 'root' });
        }
    });

    up.on('up:fragment:loaded', function (event) {
        const isFailed = up.network.config.fail(event.renderOptions.response);
        if (isFailed && event.response.status != 422) {
            // Force the fail layer or show an error alert
            event.renderOptions.failLayer = 'root';
            event.renderOptions.failTarget = '#errors';
        }
    });

    up.compiler('#list', (elem, data) => {
        // auto reload #list when server returns it in an overlay
        elem.setAttribute('up-hungry', '');
        elem.setAttribute('up-if-layer', 'subtree')

        var options = {
            valueNames: [
                { name: 'value-0', attr: 'data-search' },
                { name: 'value-1', attr: 'data-search' },
                { name: 'value-2', attr: 'data-search' },
                { name: 'value-3', attr: 'data-search' },
                { name: 'value-4', attr: 'data-search' },
                { name: 'value-5', attr: 'data-search' },
                { name: 'value-6', attr: 'data-search' },
                { name: 'value-7', attr: 'data-search' },
                { name: 'value-8', attr: 'data-search' },
                { name: 'value-9', attr: 'data-search' },
            ],
            listClass: 'values'
        };
        // scoped from elem (this #list instance), not document — an overlay's
        // #list coexists in the DOM with the base layer's own #list/search
        // while open, and a document-wide lookup would silently bind to
        // whichever one comes first
        var table = elem.querySelector("table");
        // #list names a section container, not a grid; List.js has nothing to
        // bind to without one, and this compiler is shared by every page
        if (!table) { return; }

        var search = elem.closest("main")?.querySelector("[name='search']");
        var list = new List(table, options);

        if (search) {
            list.search(search.value);
            search?.addEventListener('keyup', (event) => {
                list.search(search.value);
            });
        }

        var order = table?.dataset?.defaultSort;
        if (order) {
            list.sort(order, { order: "asc" });
            elem.querySelector("[data-sort='" + order + "']").classList.add("asc");
        }

        // lock table size in place
        table.querySelectorAll("td.fixed-width").forEach(elem => {
            var width = elem.clientWidth
            elem.style.width = width + "px"
            elem.style.maxWidth = width + "px"
        });
    });

    // Quick case switcher: arrow-key highlight + Enter through the results.
    // Compiled on the results <ul>, so it re-binds (and re-defaults the
    // highlight to the first row) after every autosubmit swap. Keydown is bound
    // on the search input, which survives the swap, so the destructor removes it.
    up.compiler('#switch-results', (elem) => {
        const input = elem.closest('#switcher')?.querySelector("input[name='search']");
        const items = () => Array.from(elem.querySelectorAll('a.switch-result'));
        const setActive = (idx) => {
            items().forEach((a, i) => a.classList.toggle('menu-focus', i === idx));
        };

        setActive(0);
        if (!input) { return; }

        const onKey = (event) => {
            const list = items();
            if (list.length === 0) { return; }
            const idx = list.findIndex(a => a.classList.contains('menu-focus'));
            if (event.key === 'ArrowDown') {
                event.preventDefault();
                setActive(Math.min(idx + 1, list.length - 1));
            } else if (event.key === 'ArrowUp') {
                event.preventDefault();
                setActive(Math.max(idx - 1, 0));
            } else if (event.key === 'Enter') {
                event.preventDefault();
                (list[idx] || list[0]).click();
            }
        };

        input.addEventListener('keydown', onKey);
        return () => input.removeEventListener('keydown', onKey);
    });

    up.compiler('select.choices:is([multiple])', (elem, data) => {
        new Choices(elem, {
            addItems: true,
            addChoices: true,
            classNames: {
                containerOuter: ['choices', 'overflow-hidden'],
                listDropdown: ['choices__list--dropdown', 'dropdown-content'],
                openState: ['overflow-visible'],
            },
            removeItems: true,
            removeItemButton: true,
            removeItemIconText: '&times;',
        });
    });

    up.compiler('select.choices:not([multiple])', (elem, data) => {
        new Choices(elem, {
            classNames: {
                containerOuter: ['choices', 'overflow-hidden'],
                listDropdown: ['choices__list--dropdown', 'dropdown-content'],
                openState: ['overflow-visible'],
            },
        });
    });
};

// --- Per-behavior compilers (CSP: no inline onclick=/onchange= attributes) -
// Each behavior gets its own up.compiler(); Unpoly runs these on the initial
// page and re-runs them on every fragment swap, so no destructors are needed
// — the listeners die with the compiled element (or its subtree) on removal.
// Style rationale: ADR 0004.

// Flips a password field between masked and plain text. Reveal button on
// password fields (see passwordField in form.templ).
up.compiler('[data-toggle-password]', (btn) => {
    btn.addEventListener('click', () => {
        const input = btn.parentElement.querySelector('input');
        input.type = input.type === 'password' ? 'text' : 'password';
    });
});

// Fills the text input in the same .join group with the current time in
// ISO-8601. "Now" button on event/task forms.
up.compiler('[data-set-now]', (btn) => {
    btn.addEventListener('click', () => {
        const input = btn.parentElement.querySelector('input');
        if (input) { input.value = new Date().toISOString(); }
    });
});

// Sets the from/to date inputs in the same form to the quick range named by
// the button's data-fill-date-range, and dispatches a change event so the
// form's up-autosubmit picks it up. Dashboard's quick-range buttons.
up.compiler('[data-fill-date-range]', (btn) => {
    btn.addEventListener('click', () => {
        const form = btn.closest('form');
        const from = form.querySelector('input[name="from"]');
        const to = form.querySelector('input[name="to"]');
        const range = btn.dataset.fillDateRange;
        const year = new Date().getFullYear();
        if (range === 'this-year') {
            from.value = `${year}-01-01`;
            to.value = `${year}-12-31`;
        } else if (range === 'last-year') {
            from.value = `${year - 1}-01-01`;
            to.value = `${year - 1}-12-31`;
        } else {
            from.value = '';
            to.value = '';
        }
        from.dispatchEvent(new Event('change', { bubbles: true }));
    });
});

// Collapses/expands a settings category. Bound on the category band row; the
// next <tbody> holds that category's data rows.
up.compiler('[data-toggle-category]', (el) => {
    el.addEventListener('click', () => {
        const band = el.closest('tbody');
        const data = band?.nextElementSibling;
        if (!data) { return; }
        data.toggleAttribute('hidden');
        band.querySelector('.chevron')?.classList.toggle('rotate-90');
    });
});

// Cycles Light -> Dark -> Auto -> Light. Current state is the data-theme
// attribute already on <html> (server-rendered, see sidebar() in layout.templ).
// No server round-trip: sets/removes data-theme and the theme cookie directly
// and swaps its own icon to match. Auto is "no cookie", not a literal value.
up.compiler('[data-theme-toggle]', (btn) => {
    const icon = btn.querySelector('i');
    btn.addEventListener('click', () => {
        const current = document.documentElement.dataset.theme;
        const next = current === 'dagobert' ? 'dagobert-dark' : current === 'dagobert-dark' ? '' : 'dagobert';

        if (next === '') {
            delete document.documentElement.dataset.theme;
            document.cookie = 'theme=; path=/; max-age=0; SameSite=Lax';
        } else {
            document.documentElement.dataset.theme = next;
            document.cookie = `theme=${next === 'dagobert-dark' ? 'dark' : 'light'}; path=/; max-age=31536000; SameSite=Lax`;
        }

        // keep in sync with themeToggleIcon + the rail's icon classes in layout.templ
        icon.className = 'ph ph-6 opacity-55 ' +
            (next === 'dagobert' ? 'ph-sun' : next === 'dagobert-dark' ? 'ph-moon' : 'ph-circle-half');
    });
});

up.compiler('[data-remove-self]', (el) => {
    el.addEventListener('click', () => el.remove());
});

up.compiler('[data-copy-reveal-key]', (el) => {
    el.addEventListener('click', () => navigator.clipboard.writeText(document.getElementById('reveal-key').value));
});

up.compiler('[data-stop-propagation]', (el) => {
    el.addEventListener('click', (event) => event.stopPropagation());
});

// Shows the "Options" field only for the "select" custom attribute type.
up.compiler('[data-toggle-custom-options]', (sel) => {
    sel.addEventListener('change', () => {
        document.getElementById('custom-options').style.display = sel.value === 'select' ? '' : 'none';
    });
});

// Fills the case form's case-level defaults from the picked template's inline
// data-* attributes, with no server roundtrip. "Create from template" <select>
// on the new-case form (see CasesOne).
up.compiler('[data-apply-case-template]', (select) => {
    select.addEventListener('change', () => {
        const opt = select.options[select.selectedIndex];
        const form = select.form;
        form.elements['Classification'].value = opt.dataset.classification || '';
        form.elements['Severity'].value = opt.dataset.severity || '';
        form.elements['Summary'].value = opt.dataset.summary || '';
    });
});

up.compiler('[data-apply-template-name]', (el) => {
    el.addEventListener('change', () => {
        document.querySelector('input[name=Name]').value =
            el.options[el.selectedIndex].text + ' (Template)';
    });
});

// Faithful non-eval replacement for up-on-accepted="..." (Unpoly's internal
// new Function() eval, blocked by CSP with no 'unsafe-eval'). Unpoly fires
// up:layer:accepted on the link that opened the layer, same as what
// up-on-accepted evaluates internally.
up.compiler('[data-on-accepted=reload-list]', (link) => {
    link.addEventListener('up:layer:accepted', () => up.reload('#list'));
});
up.compiler('[data-on-accepted=reload-main-root]', (link) => {
    link.addEventListener('up:layer:accepted', () => up.reload('main', { layer: 'root' }));
});
up.compiler('[data-on-accepted=goto-cases]', (link) => {
    link.addEventListener('up:layer:accepted', () => up.navigate({ url: '/cases/', layer: 'root' }));
});

// Row navigation for settings-style tables: clicking a row (but not a link or
// button inside it) navigates to its data-href.
up.compiler('table:has([data-href])', (table) => {
    table.addEventListener('click', (event) => {
        const row = event.target.closest('[data-href]');
        if (row && !event.target.closest('a, button')) { location.assign(row.dataset.href); }
    });
});

// showToast renders a success toast into the root #errors section: a seal
// glyph, a monospace caption with the time, and the message. Matches the markup
// of the server-rendered ToastError/ToastWarning.
function showToast(message) {
    const container = document.querySelector('#errors');
    if (!container) { return; }
    container.className = 'toast toast-top toast-center z-20';

    const seal = document.createElement('div');
    seal.className = 'seal';
    seal.setAttribute('role', 'status');
    seal.setAttribute('aria-live', 'polite');
    seal.onclick = () => seal.remove();
    seal.innerHTML = '<span class="seal-mark" aria-hidden="true"><i class="ph ph-seal-check"></i></span>'
        + '<div class="seal-body"><div class="label-micro"></div>'
        + '<div class="seal-msg"></div></div>';
    seal.querySelector('.label-micro').textContent =
        'Recorded · ' + new Date().toLocaleTimeString([], { hour12: false });
    seal.querySelector('.seal-msg').textContent = message;
    container.appendChild(seal);

    setTimeout(() => seal.remove(), 4000);
}

// --- Fragment compilers ---------------------------------------------------
// Registered at module scope (not inside the onload handler) so they run before
// Unpoly boots and therefore apply to the initial page as well as later
// fragment swaps. Unpoly 3.11+ no longer executes <script> elements inside
// swapped fragments, so the inline scripts these replace would otherwise never
// run. See https://unpoly.com/legacy-scripts.

// File inputs marked [data-fill] copy the picked file's basename into the named
// form field; those marked [data-hash] compute the file's SHA-1 into the form's
// Hash field (evidence + malware upload forms, report uploads).
up.compiler('input[type=file][data-fill], input[type=file][data-hash]', (input) => {
    const onChange = () => {
        const form = input.form;
        if (input.dataset.fill) {
            const target = form.querySelector('input[name="' + input.dataset.fill + '"]');
            if (target) { target.value = input.value.replace(/.*(\/|\\)/, ''); }
        }
        if (input.dataset.hash !== undefined && input.files[0]) {
            hashfile(input.files[0], form);
            const size = form.querySelector('input[name="Size"]');
            if (size) { size.value = input.files[0].size; }
        }
    };
    input.addEventListener('change', onChange);
    return () => input.removeEventListener('change', onChange);
});

// Markdown live-preview editor (Vditor, instant-render mode) for textareas
// marked [data-markdown] (note Description, case Summary). The textarea stays
// in the DOM as the hidden form field; the editor syncs into it on input.
// If Vditor fails to load, the plain textarea remains usable.
up.compiler('textarea[data-markdown]', (textarea) => {
    if (typeof Vditor === 'undefined') { return; }
    const holder = document.createElement('div');
    textarea.insertAdjacentElement('afterend', holder);
    textarea.hidden = true;
    const editor = new Vditor(holder, {
        mode: 'ir',
        lang: 'en_US',
        icon: null, // toolbar is hidden; skips loading dist/js/icons/*.js
        cdn: '/public/assets/vditor-3.11.2',
        value: textarea.value,
        toolbar: [],
        cache: { enable: false },
        preview: {
            hljs: { enable: false },
            theme: { current: 'dagobert', path: '/public/assets/vditor-3.11.2/dist/css/content-theme' },
        },
        // a floor only; CSS grows the editor into whatever room the drawer has
        // spare, see .form-control:has(> .vditor)
        minHeight: 320,
        input: (value) => { textarea.value = value; },
    });
    return () => {
        editor.destroy();
        holder.remove();
        textarea.hidden = false;
    };
});

// Collapse long markdown previews in tables. CSS caps .markdown-preview at a
// generous max-height; when the content actually overflows, add the fade-out
// mask and a show more/less toggle below it.
up.compiler('.markdown-preview', (elem) => {
    if (elem.scrollHeight <= elem.clientHeight) { return; }
    elem.classList.add('overflowing');
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'btn btn-ghost btn-xs';
    btn.textContent = 'Show more';
    btn.addEventListener('click', () => {
        const expanded = elem.classList.toggle('expanded');
        btn.textContent = expanded ? 'Show less' : 'Show more';
    });
    elem.insertAdjacentElement('afterend', btn);
    return () => btn.remove();
});

// --- The lateral-movement sheet (VisNetwork) -----------------------------
//
// Nodes and edges arrive from the server in [up-data]; vis-network is loaded on
// demand so its bundle only ships on this page. The drawn network is held here
// rather than on the element so the toolbar — which lives in the page header,
// outside the swapped #list fragment — can reach whatever is currently drawn.
let graphNetwork = null;
let graphStill = false;

// vis draws to canvas, so every colour has to arrive as a resolved string, and
// getComputedStyle hands back custom properties unresolved (`color-mix(…)` as
// text). Each one is therefore round-tripped through a throwaway element and
// read back off `color`, which is always a resolved colour.
const GRAPH_COLORS = ['node', 'node-dim', 'severe', 'edge', 'edge-dim', 'signal', 'label', 'halo',
    'paper', 'ring', 'severe-ring', 'severe-wash'];

// The palette is filled in place rather than returned fresh: a node's renderer
// closes over this object and reads it at draw time, so re-resolving into the
// same object plus a redraw is all a theme change costs. Handing the nodes a
// new closure instead would mean replacing every node in the DataSet, and
// vis keeps the shape instance it first built.
function graphPalette(elem, into = {}) {
    const probe = document.createElement('span');
    probe.style.display = 'none';
    elem.appendChild(probe);
    for (const name of GRAPH_COLORS) {
        probe.style.color = `var(--graph-${name})`;
        into[name] = getComputedStyle(probe).color;
    }
    probe.remove();
    return into;
}

// A node is a struck token: the glyph sits on its own disc of sheet paper with
// a hairline ring, so an edge stops at the token's edge instead of running
// through the drawing. vis draws edges first and nodes second, so an opaque
// fill in the node pass is what cuts the line — which is why this is a custom
// shape rather than the built-in `icon` (glyph only, nothing behind it).
//
// An asset is the subject and an indicator is evidence attached to it, so the
// two are separated by weight rather than by hue: both are ink, and the only
// coloured mark on the sheet is a severe state, drawn as the same struck stamp
// the confirm dialog uses — a 45% ring over a 6% wash.
function graphNodes(rows, c) {
    return rows.map((n) => {
        const asset = n.kind !== 'indicator';
        const radius = asset ? 20 : 14;
        const glyph = asset ? 22 : 15;
        // read out of the palette as it draws, never captured here: the whole
        // point of the palette being one mutated object is that a theme change
        // is a redraw rather than a rebuild of every node
        const inkNow = () => (n.severe ? c.severe : (asset ? c.node : c['node-dim']));

        return {
            id: n.id,
            label: n.label,
            url: n.url,
            title: graphTip(n),
            shape: 'custom',
            ctxRenderer: ({ ctx, x, y, state: { selected, hover } }) => ({
                drawNode() {
                    ctx.save();
                    ctx.beginPath();
                    ctx.arc(x, y, radius, 0, 2 * Math.PI);
                    // paper first, then the wash on top of it: a 6% tint laid
                    // straight onto the canvas would let the edge show through
                    ctx.fillStyle = c.paper;
                    ctx.fill();
                    if (n.severe) {
                        ctx.fillStyle = c['severe-wash'];
                        ctx.fill();
                    }
                    ctx.lineWidth = 1;
                    ctx.strokeStyle = (selected || hover) ? c.signal
                        : (n.severe ? c['severe-ring'] : c.ring);
                    ctx.stroke();

                    ctx.fillStyle = inkNow();
                    ctx.textAlign = 'center';
                    ctx.textBaseline = 'middle';
                    if (n.icon) {
                        ctx.font = `${glyph}px 'Phosphor'`;
                        ctx.fillText(n.icon, x, y);
                    } else {
                        // an unmapped type has no glyph to draw; a struck centre
                        // beats the missing-character box the icon font renders
                        ctx.beginPath();
                        ctx.arc(x, y, asset ? 5 : 4, 0, 2 * Math.PI);
                        ctx.fill();
                    }
                    ctx.restore();
                },
                // A custom shape draws its own label or gets none. The word
                // always takes plain ink — the glyph is what carries the severe
                // hue, the same split the status dot makes in the grids — and
                // the halo is a stroked outline rather than a filled box, which
                // would put a second rectangle on a sheet that is already ruled.
                drawExternalLabel() {
                    ctx.save();
                    ctx.font = `${asset ? 11 : 10}px 'JetBrains Mono Variable', ui-monospace, monospace`;
                    ctx.textAlign = 'center';
                    ctx.textBaseline = 'top';
                    ctx.lineWidth = 4;
                    ctx.lineJoin = 'round';
                    ctx.strokeStyle = c.halo;
                    ctx.strokeText(n.label, x, y + radius + 6);
                    ctx.fillStyle = c.label;
                    ctx.fillText(n.label, x, y + radius + 6);
                    ctx.restore();
                },
                nodeDimensions: { width: 2 * radius, height: 2 * radius },
            }),
        };
    });
}

// Framing the sheet, from the node positions rather than through vis's own fit.
// That fit stops at scale 1 — a six-node case then sits as a small drawing in a
// large panel — and it measures bounding boxes that neither match a custom
// shape nor include the labels drawn beside it, so it leaves slack on every
// side and still crops the words. Working from the coordinates makes the margin
// an explicit number: GRAPH_PAD is the token plus the room a label needs.
const GRAPH_PAD = 90;
const GRAPH_MAX_ZOOM = 1.5;

function graphFit(network, animate) {
    const positions = network.getPositions();
    const ids = Object.keys(positions);
    if (!ids.length) { return; }

    const xs = ids.map((id) => positions[id].x);
    const ys = ids.map((id) => positions[id].y);
    const box = { l: Math.min(...xs), r: Math.max(...xs), t: Math.min(...ys), b: Math.max(...ys) };
    const view = network.body.container;
    const scale = Math.min(
        GRAPH_MAX_ZOOM,
        view.clientWidth / (box.r - box.l + 2 * GRAPH_PAD),
        view.clientHeight / (box.b - box.t + 2 * GRAPH_PAD),
    );

    network.moveTo({
        scale,
        position: { x: (box.l + box.r) / 2, y: (box.t + box.b) / 2 },
        animation: animate ? { duration: 400, easingFunction: 'easeOutQuint' } : false,
    });
}

function graphEdges(rows, c) {
    return rows.map((e) => (e.kind === 'observation'
        ? { from: e.from, to: e.to, dashes: [2, 4], width: 1, color: { color: c['edge-dim'], highlight: c.signal, hover: c.signal } }
        : { from: e.from, to: e.to, width: 1.5, color: { color: c.edge, highlight: c.signal, hover: c.signal } }));
}

// graphTip builds a node's hover card as DOM. It is deliberately not markup
// from the server: vis-network writes a string title straight into innerHTML,
// which would make every asset name and indicator value an injection point.
function graphTip(node) {
    const tip = document.createElement('div');
    tip.className = node.severe ? 'graph-tip graph-tip-severe' : 'graph-tip';

    const head = document.createElement('div');
    head.className = 'graph-tip-head';
    head.textContent = node.tip.head;
    tip.appendChild(head);

    for (const [label, value] of node.tip.rows || []) {
        const row = document.createElement('div');
        row.className = 'graph-tip-row';
        const key = document.createElement('span');
        key.className = 'graph-tip-label';
        key.textContent = label;
        const val = document.createElement('span');
        val.textContent = value;
        row.append(key, val);
        tip.appendChild(row);
    }

    if (node.url) {
        const hint = document.createElement('div');
        hint.className = 'graph-tip-hint';
        hint.textContent = 'Double-click to open';
        tip.appendChild(hint);
    }
    return tip;
}

up.compiler('#mynetwork', (elem, data) => {
    const status = elem.parentElement.querySelector('[data-graph-status]');
    graphStill = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    let network = null;
    let watchers = [];
    loadScript('/public/assets/vis-network-10.1.0.min.js', () => window.vis && window.vis.Network).then(() => {
        const c = graphPalette(elem);
        const nodesDS = new vis.DataSet(graphNodes(data.nodes, c));
        const edgesDS = new vis.DataSet(graphEdges(data.edges, c));

        const options = {
            // Hovering a node lights its neighbourhood in the signal colour —
            // which is what the signal colour is for. The width deltas are off
            // so the highlight changes the colour of a line and not its weight:
            // vis's defaults thicken it to three times a hairline.
            edges: { smooth: { type: 'continuous', roundness: 0.25 }, selectionWidth: 0, hoverWidth: 0 },
            physics: {
                solver: 'repulsion',
                repulsion: { centralGravity: 0.25, springLength: 150, nodeDistance: 175, damping: 0.15 },
                minVelocity: 0.75,
                // settle before the first paint rather than in front of the
                // analyst: a graph that swims into place for two seconds reads
                // as a toy, and under reduced motion it is not allowed at all
                stabilization: { enabled: true, iterations: 400, updateInterval: 400, fit: true },
            },
            // the container is focusable, so arrow keys pan the sheet the same
            // way dragging does — without stealing them from the rest of the page
            interaction: { hover: true, tooltipDelay: 200, keyboard: { enabled: true, bindToWindow: false }, navigationButtons: false },
            layout: { randomSeed: 7 },
        };

        network = new vis.Network(elem, { nodes: nodesDS, edges: edgesDS }, options);
        graphNetwork = network;

        // The theme toggle rewrites data-theme in place, with no round trip, so
        // a canvas keeps whatever palette it was drawn with — the one surface in
        // the product that CSS cannot restyle for free. Repaint on both the ways
        // the theme can change: the explicit choice, and the OS preference while
        // the choice is "auto".
        const repaint = () => {
            graphPalette(elem, c);
            // the nodes read the palette as they draw; the edges carry their
            // colours as data and have to be handed the new ones
            edgesDS.update(graphEdges(data.edges, c));
            network.redraw();
        };
        const observer = new MutationObserver(repaint);
        observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
        const os = window.matchMedia('(prefers-color-scheme: dark)');
        os.addEventListener('change', repaint);
        watchers = [() => observer.disconnect(), () => os.removeEventListener('change', repaint)];

        const reveal = () => {
            if (status) { status.remove(); }
            elem.classList.add('is-drawn');
        };
        network.once('stabilizationIterationsDone', () => {
            reveal();
            graphFit(network, false);
            // reduced motion means the sheet stays exactly where it settled
            if (graphStill) { setGraphFrozen(document.querySelector('[data-graph-action="freeze"]'), true); }
        });
        // a graph that never uncovers itself is worse than one that settles on
        // screen, so the reveal does not depend on that event alone
        setTimeout(reveal, 2000);

        network.on('hoverNode', () => { elem.style.cursor = 'pointer'; });
        network.on('blurNode', () => { elem.style.cursor = ''; });

        // Double-click, not click: a single click is how you pick a node up and
        // drag it, and opening a drawer under that would fight the arranging.
        network.on('doubleClick', (params) => {
            const node = params.nodes.length && network.body.data.nodes.get(params.nodes[0]);
            if (!node || !node.url) { return; }
            // The record's own edit drawer, the same one its row opens. Saving
            // redirects to the record's section list, and reaching that location
            // is what accepts the layer — hence the accept location below.
            //
            // Unpoly then navigates the PARENT layer to that same location. On a
            // list page that is invisible and useful: it is how the grid behind
            // the drawer refreshes itself after a save. Here it is exactly
            // wrong, because the page behind the drawer is not the list the
            // record belongs to — which is why editing a node used to leave the
            // analyst looking at the indicators table. So that one render is
            // vetoed and the sheet is drawn again in its place.
            //
            // `history: false` keeps the drawer out of the address bar, so
            // there is no overlay location to be restored over the sheet either.
            const sheet = up.layer.location;
            const list = node.url.slice(0, node.url.lastIndexOf('/') + 1);
            up.layer.open({
                url: node.url,
                mode: 'drawer',
                history: false,
                acceptLocation: list,
                onAccepted: () => {
                    const restore = (event) => {
                        if (!event.response.url.endsWith(list)) { return; }
                        up.off('up:fragment:loaded', restore);
                        // skip(), not preventDefault(): both drop the render,
                        // but preventing it rejects Unpoly's own render job with
                        // an AbortError nobody is there to catch, and a console
                        // error on every save is not a thing this page should do
                        event.skip();
                        up.render({ target: '#graph', url: sheet, layer: 'root' });
                    };
                    up.on('up:fragment:loaded', restore);
                    // disarm if that navigation never comes, so the guard can
                    // never veto an unrelated render later on
                    setTimeout(() => up.off('up:fragment:loaded', restore), 2000);
                },
            }).catch(() => {});
        });
    });

    return () => {
        watchers.forEach((off) => off());
        if (network) { network.destroy(); }
        if (graphNetwork === network) { graphNetwork = null; }
    };
});

// The graph toolbar lives in the page header, outside the fragment the graph is
// swapped in with, so it acts on whatever is currently drawn rather than owning it.
up.compiler('[data-graph-action]', (btn) => {
    const onClick = () => {
        if (!graphNetwork) { return; }
        if (btn.dataset.graphAction === 'fit') {
            graphFit(graphNetwork, !graphStill);
        } else {
            setGraphFrozen(btn, btn.getAttribute('aria-pressed') !== 'true');
        }
    };
    btn.addEventListener('click', onClick);
    return () => btn.removeEventListener('click', onClick);
});

// Freezing stops the simulation so a node stays where it was dragged. The state
// is on the button rather than in a variable: it is the thing that has to show it.
function setGraphFrozen(btn, frozen) {
    if (!btn) { return; }
    btn.setAttribute('aria-pressed', String(frozen));
    btn.querySelector('i').className = `ph ph-5 inline-block mr-1 ph-lock-simple${frozen ? '' : '-open'}`;
    btn.querySelector('[data-graph-label]').textContent = frozen ? 'Frozen' : 'Freeze';
    if (graphNetwork) { graphNetwork.setOptions({ physics: { enabled: !frozen } }); }
}

// Event timeline histogram (EventsMany). Bucketed counts arrive in [up-data];
// vis-timeline is loaded on demand.
up.compiler('#histogram', (elem, data) => {
    loadScript('/public/assets/vis-timeline-8.5.1.min.js', () => window.vis && window.vis.Graph2d).then(() => {
        const options = {
            style: "bar",
            barChart: { align: "center" },
            dataAxis: { visible: false },
            drawPoints: false,
            height: "150px",
            orientation: "bottom",
            moment: (date) => vis.moment(date).utc(),
        };
        new vis.Graph2d(elem, new vis.DataSet(data), options);
    });
});

// --- Helpers invoked from inline on* handlers / the compilers above -------

// hashfile computes the SHA-1 of the picked file and writes it (hex) into the
// form's Hash field.
function hashfile(file, form) {
    readbinaryfile(file)
        .then((buf) => crypto.subtle.digest('SHA-1', new Uint8Array(buf)))
        .then((digest) => {
            const hash = form.querySelector('input[name="Hash"]');
            if (hash) { hash.value = Uint8ArrayToHexString(new Uint8Array(digest)); }
        });
}

function readbinaryfile(file) {
    return new Promise((resolve, reject) => {
        const fr = new FileReader();
        fr.onload = () => resolve(fr.result);
        fr.onerror = reject;
        fr.readAsArrayBuffer(file);
    });
}

function Uint8ArrayToHexString(arr) {
    let hex = '';
    for (let i = 0; i < arr.length; i++) {
        hex += arr[i].toString(16).padStart(2, '0');
    }
    return hex;
}

// loadScript appends a <script> for src unless isReady() reports the needed
// global is already present. Re-appending re-runs the (cached) bundle, so moving
// between the vis-network and vis-timeline pages re-establishes the right
// window.vis API.
function loadScript(src, isReady) {
    if (isReady && isReady()) { return Promise.resolve(); }
    return new Promise((resolve, reject) => {
        const s = document.createElement('script');
        s.src = src;
        s.onload = resolve;
        s.onerror = reject;
        document.head.appendChild(s);
    });
}