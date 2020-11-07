
function updateButtonsStatus() {
    let fileCount = 0;
    let folderCount = 0;
    $('.file-row').each(function () {
        if ($(this).find('input').is(':checked')) {
            const i = $(this).find('.i')
            if (i.hasClass('fa-file')) {
                fileCount++;
            } else if (i.hasClass('fa-folder')) {
                folderCount++;
            }
        }
    })

    $('#download').prop('disabled', folderCount > 0 || fileCount === 0)
    $('#delete').prop('disabled', folderCount === 0 && fileCount === 0)
    $('#move').prop('disabled', (folderCount + fileCount) !== 1)
    $('#archive').prop('disabled', folderCount === 0 && fileCount === 0)
}

$('.selectAll').change(function () {
    const checked = $(this).is(':checked')
    $('.select').each(function (i) {
        $(this).prop('checked', checked)
    })
    updateButtonsStatus();
})

$('.select').change(updateButtonsStatus)
updateButtonsStatus()