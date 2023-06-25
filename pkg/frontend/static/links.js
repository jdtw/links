async function removeLink(remove, target) {
  if (!confirm(`Remove ${remove}?`)) return;
  let resp = await fetch(`/rm/${remove}`, { method: "DELETE" });
  if (!resp.ok) {
    let err = await resp.text();
    alert(err)
    return
  }
  target.closest("tr").remove();
}

function editLink(link) {
  const form = document.getElementById("link-form");
  const [linkInput, uriInput] = form.querySelectorAll("input");
  linkInput.value = link;
  uriInput.value = document.getElementById(link).href;
  uriInput.focus();
}

const links = document.getElementById("links");
links.addEventListener("click", async (event) => {
  const remove = event.target.dataset.remove;
  if (remove) {
    return removeLink(remove, event.target);
  }
  const edit = event.target.dataset.edit;
  if (edit) {
    return editLink(edit);
  }
});