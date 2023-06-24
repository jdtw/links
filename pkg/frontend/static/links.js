const links = document.getElementById("links");
links.addEventListener("click", async (event) => {
  const rm = event.target.dataset.remove;
  if (!rm) return;
  if (!confirm(`Remove ${rm}?`)) return;
  let resp = await fetch(`/rm/${rm}`, { method: "DELETE" });
  if (!resp.ok) {
    let err = await resp.text();
    alert(err)
    return
  }
  event.target.closest("tr").remove();
});