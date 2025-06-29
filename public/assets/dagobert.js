onload = (event) => {
    // up.log.enable();
    up.log.disable();
    up.network.config.autoCache = (request) => false;
    up.network.config.wrapMethod = false;
    up.layer.config.drawer.position = 'right';
    up.layer.config.drawer.size = 'large';

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