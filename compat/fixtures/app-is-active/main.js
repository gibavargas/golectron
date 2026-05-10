const { app } = require('electron');

async function main() {
  await app.whenReady();
  const supported = typeof app.isActive === 'function';
  console.log(JSON.stringify({
    platform: process.platform,
    supported,
    active: supported ? app.isActive() : false
  }));
  app.quit();
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
  app.quit();
});
