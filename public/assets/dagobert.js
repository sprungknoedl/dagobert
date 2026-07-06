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
        var table = document.querySelector("#list table");
        var search = document.querySelector("[name='search']");
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
            document.querySelector("[data-sort='" + order + "']").classList.add("asc");
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

// applyCaseTemplate fills the case form's case-level defaults from the picked
// template's inline data-* attributes, with no server roundtrip. Wired to the
// "Create from template" <select> on the new-case form (see CasesOne).
function applyCaseTemplate(select) {
    const opt = select.options[select.selectedIndex];
    const form = select.form;
    form.elements['Classification'].value = opt.dataset.classification || '';
    form.elements['Severity'].value = opt.dataset.severity || '';
    form.elements['Summary'].value = opt.dataset.summary || '';
}

// --- Delegated dispatch (CSP: no inline onclick=/onchange= attributes) -----
// Document-level listeners need no up.compiler() registration/teardown —
// they're live for the initial page and all later Unpoly fragment swaps
// without rebinding.
const actions = {
    togglePassword,
    setNow,
    toggleCustomOptions,
    applyCaseTemplate,
    toggleCategory,
    removeSelf: (el) => el.remove(),
    copyRevealKey: () => navigator.clipboard.writeText(document.getElementById('reveal-key').value),
    applyTemplateName: (el) => {
        document.querySelector('input[name=Name]').value =
            el.options[el.selectedIndex].text + ' (Template)';
    },
    stopPropagation: (el, event) => event.stopPropagation(),
};

document.addEventListener('click', (event) => {
    const el = event.target.closest('[data-onclick]');
    if (el) { actions[el.dataset.onclick]?.(el, event); }

    const link = event.target.closest('[data-href]');
    if (link && !event.target.closest('a, button')) { location.assign(link.dataset.href); }
});

document.addEventListener('change', (event) => {
    const el = event.target.closest('[data-onchange]');
    if (el) { actions[el.dataset.onchange]?.(el, event); }
});

// Faithful non-eval replacement for up-on-accepted="..." (Unpoly's internal
// new Function() eval, blocked by CSP with no 'unsafe-eval'). Unpoly fires
// up:layer:accepted on the link that opened the layer, same as what
// up-on-accepted evaluates internally.
const onAccepted = {
    'reload-list': () => up.reload('#list'),
    'reload-main-root': () => up.reload('main', { layer: 'root' }),
    'goto-cases': () => up.navigate({ url: '/cases/', layer: 'root' }),
};
up.compiler('[data-up-accepted]', (link) => {
    const fn = onAccepted[link.dataset.upAccepted];
    if (fn) { link.addEventListener('up:layer:accepted', fn); }
});

// showToast renders a transient success toast into the root #errors section,
// matching the daisyUI markup of the server-rendered error/warning toasts.
function showToast(message) {
    const container = document.querySelector('#errors');
    if (!container) { return; }
    container.className = 'toast toast-top toast-center z-20';

    const alert = document.createElement('div');
    alert.className = 'alert alert-success w-[42rem] m-4';
    alert.setAttribute('role', 'alert');
    alert.onclick = () => alert.remove();
    alert.innerHTML = '<i class="ph ph-check-circle text-3xl"></i><div>'
        + '<h3 class="font-bold">Success</h3>'
        + '<div class="text-xs"></div></div>';
    alert.querySelector('.text-xs').textContent = message;
    container.appendChild(alert);

    setTimeout(() => alert.remove(), 4000);
}

// toggleCategory collapses/expands a settings category. Bound via inline onclick
// on the category band row; the next <tbody> holds that category's data rows.
function toggleCategory(el) {
    const band = el.closest('tbody');
    const data = band?.nextElementSibling;
    if (!data) { return; }
    data.toggleAttribute('hidden');
    band.querySelector('.chevron')?.classList.toggle('rotate-90');
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
        minHeight: 120,
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

// Lateral-movement network graph (VisNetwork). Nodes/edges/groups arrive from
// the server in [up-data]; vis-network is loaded on demand so it only ships on
// this page.
up.compiler('#mynetwork', (elem, data) => {
    loadScript('/public/assets/vis-network-10.1.0.min.js', () => window.vis && window.vis.Network).then(() => {
        const options = {
            edges: {
                color: { color: "oklch(72% 0.13 80)", highlight: "oklch(70% 0.15 70)" },
                smooth: { forceDirection: "vertical" },
            },
            nodes: {
                shape: "icon",
                margin: 10,
                font: { color: "oklch(25% 0.02 60)", background: "oklch(97.5% 0.01 90)" },
                icon: { face: "'Phosphor'" },
            },
            groups: data.groups,
            physics: {
                repulsion: { centralGravity: 0.25, springLength: 150, nodeDistance: 175, damping: 0.15 },
                minVelocity: 0.75,
                solver: "repulsion",
            },
        };
        new vis.Network(elem, { nodes: new vis.DataSet(data.nodes), edges: new vis.DataSet(data.edges) }, options);
    });
});

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

// togglePassword flips a password field between masked and plain text. Wired via
// inline onclick on the reveal button (see passwordField in form.templ).
function togglePassword(btn) {
    const input = btn.parentElement.querySelector('input');
    input.type = input.type === 'password' ? 'text' : 'password';
}

// setNow fills the text input in the same .join group with the current time in
// ISO-8601. Wired via inline onclick on the "Now" button (event/task forms).
function setNow(btn) {
    const input = btn.parentElement.querySelector('input');
    if (input) { input.value = new Date().toISOString(); }
}

// toggleCustomOptions shows the "Options" field only for the "select" custom
// attribute type. Wired via inline onchange on the type <select>.
function toggleCustomOptions(sel) {
    document.getElementById('custom-options').style.display = sel.value === 'select' ? '' : 'none';
}

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