function updateButtonsStatus() {
  console.log("update buttons status");
  let fileCount = 0;
  let folderCount = 0;
  $(".file-row").each(function () {
    if ($(this).find("input").is(":checked")) {
      const i = $(this).find("i");
      if (i.attr("isdir") == "true") {
        folderCount++;
      } else {
        fileCount++;
      }
    }
  });
  console.log("file count:", fileCount, ", folder count:", folderCount);
  $("#download").prop("disabled", folderCount > 0 || fileCount === 0);
  $("#delete").prop("disabled", folderCount === 0 && fileCount === 0);
  $("#move").prop("disabled", folderCount + fileCount !== 1);
  $("#archive").prop("disabled", folderCount === 0 && fileCount === 0);
}

function download(name, url) {
  var aDom = document.createElement("a");
  var evt = document.createEvent("HTMLEvents");
  evt.initEvent("click", false, false);
  aDom.download = name;
  aDom.href = url;
  aDom.dispatchEvent(evt);
  aDom.click();
}

function isValidPath(path) {
  if (path.length == 0) {
    return false;
  }

  const arr = [":", "*", "?", "\"", "<", ">", "|"];
  for (let a of arr) {
    if (path.indexOf(a) >= 0) {
      return false;
    }
  }

  return true;
}

function getSelectFiles() {
  let selectFiles = [];
  $(".file-row").each(function () {
    if ($(this).find("input").is(":checked")) {
      let href = $(this).find("a").attr("href");
      if (href.indexOf("?") >= 0) {
        href = href.substr(0, href.indexOf("?"));
      }
      selectFiles.push(href);
    }
  });
  return selectFiles;
}

$(".select-all").change(function () {
  const checked = $(this).is(":checked");
  $(".select").each(function (i) {
    $(this).prop("checked", checked);
  });
  updateButtonsStatus();
});

$(".select").change(updateButtonsStatus);

$("#download").click(function () {
  $(".file-row").each(function () {
    if ($(this).find("input").is(":checked")) {
      const a = $(this).find("a");
      download(a.text(), a.attr("href"));
    }
  });
});

$("#delete").click(function () {
  if (!confirm("确定要删除选中的文件吗？")) {
    return;
  }
  const selectFiles = getSelectFiles();
  fetch("/delete", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(selectFiles),
  })
    .then((response) => response.json())
    .then(function (response) {
      if (response.err) {
        alert(response.err);
      } else {
        location.reload();
      }
    })
    .catch(function (err) {
      alert(err);
    });
});

$("#move").click(function () {
  let currentdir = window.location.pathname;
  let files = getSelectFiles()
  if (files.length > 0) {
    let filename = decodeURI(files[0])
    $(".frompath").val(filename);
    $("#moveInput").val(filename);
  }
  return true;
});

$("#rename").click(function () {
  let name = window.location.pathname;
  let start =
    (name.lastIndexOf("\\") >= 0
      ? name.lastIndexOf("\\")
      : name.lastIndexOf("/")) + 1;
  $("#renameInput").val(name.substr(start));
  return true;
});

$("#filter").bind("input propertychange", function () {
  var text = $(this).val().toLowerCase();
  $(".file-row").each(function () {
    const i = $(this).find(".filename");
    const filename = i.text().toLowerCase();
    if (filename.indexOf(text) >= 0) {
      $(this).show();
    } else {
      $(this).hide();
    }
  });
});

$("#archive").click(function () {
  $("#pathlist").val(JSON.stringify(getSelectFiles()));
  $("#archiveInput").val(
    "files-" + new Date().toISOString().replace(/:/g, "") + ".zip"
  );
  return true;
});
$("#archiveOk").click(function () {
  $("#archiveSubmit").click();
});
$("#moveOk").click(function () {
  $("#moveSubmit").click();
});
$("#newOk").click(function () {
  $("#newSubmit").click();
});
$("#uploadOk").click(function () {
  $("#uploadSubmit").click();
});

updateButtonsStatus();

function getUrlParams() {
  const result = {};
  let url = window.location.href;
  const pos = url.indexOf("?");
  if (pos === -1) {
    return result;
  }
  const params = url.slice(pos + 1);
  const ls = params.split("&");
  for (const item of ls) {
    const keyvalue = item.split("=");
    if (keyvalue.length == 2) {
      if (keyvalue[0] === "s") {
        result["type"] = keyvalue[1];
      } else if (keyvalue[0] === "o") {
        result["order"] = keyvalue[1];
      }
    }
  }
  return result;
}

(function updateOrderStatus() {
  const params = getUrlParams();
  if (params.type && params.order) {
    const faSort = params.order === "asc" ? "fa-sort-asc" : "fa-sort-desc";
    const tag = $(`thead .table-header-${params.type} i`);
    tag.removeClass("fa-sort fa-sort-desc fa-sort-asc");
    tag.addClass(faSort);
  }
})();

$("thead .table-header-item label").click(function () {
  let url = window.location.href;

  let type = $(this).html().trim().toLowerCase().split(" ")[0];
  let order = "desc";
  const params = getUrlParams();
  if (params.order && params.order.length) {
    order = params.order === "desc" ? "asc" : "desc";
  }

  const pos = window.location.href.indexOf("?");
  if (pos === -1) {
    window.location.href = window.location.href + `?s=${type}&o=${order}`;
  } else {
    window.location.href =
      window.location.href.substr(0, pos) + `?s=${type}&o=${order}`;
  }
});
