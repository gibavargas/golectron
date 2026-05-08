window.addEventListener('DOMContentLoaded', async () => {
  const result = await window.fixture.ping();
  document.querySelector('#status').textContent = result;
});
