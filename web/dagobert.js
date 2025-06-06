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

        let table = new DataTable("#list table", {
            paging: false,
            searching: true,
            scrollX: true,
            typeDetect: false,
            fixedHeader: true,
            layout: { topStart: null, topEnd: null, bottomStart: null, bottomEnd: null },
            language: {
                emptyTable: 'No data available in table',
                zeroRecords: 'No records to display'
            }
        });

        if ($("[name='search']").length) {
            table.search($("[name='search']").val()).draw();
            $("[name='search']").on('keyup', function() {
                table.search($(this).val()).draw();
            });
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