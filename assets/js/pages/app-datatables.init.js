document.addEventListener("DOMContentLoaded", function () {
    var dom =
        "<'row align-items-center justify-content-between flex-nowrap g-2 app-datatable-toolbar'<'col-auto'l><'col-auto'f>>" +
        "<'row'<'col-sm-12'tr>>" +
        "<'row align-items-center justify-content-between'<'col-sm-auto'i><'col-sm-auto'p>>";

    var language = {
        lengthMenu: "Afficher _MENU_ lignes",
        search: "Rechercher :",
        info: "Affichage de _START_ à _END_ sur _TOTAL_ entrées",
        infoEmpty: "Affichage de 0 à 0 sur 0 entrée",
        infoFiltered: "(filtré à partir de _MAX_ entrées au total)",
        zeroRecords: "Aucune entrée correspondante trouvée",
        paginate: {
            first: "Premier",
            last: "Dernier",
            next: "Suivant",
            previous: "Précédent",
        },
    };

    var baseOptions = {
        responsive: true,
        dom: dom,
        order: [[0, "desc"]],
        pageLength: 10,
        lengthMenu: [10, 25, 50, 100],
        language: language,
    };

    if (document.getElementById("requisitions-datatable")) {
        new DataTable("#requisitions-datatable", Object.assign({}, baseOptions, {
            columnDefs: [{ orderable: false, targets: -1 }],
            language: Object.assign({}, language, {
                emptyTable: "Aucune réquisition disponible",
            }),
        }));
    }

    if (document.getElementById("reports-datatable")) {
        new DataTable("#reports-datatable", Object.assign({}, baseOptions, {
            language: Object.assign({}, language, {
                emptyTable: "Aucun décaissement pour cette période",
            }),
        }));
    }
});
