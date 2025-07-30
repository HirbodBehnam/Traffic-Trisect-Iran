let capturedLinks = [];

// Listen for download events
chrome.downloads.onCreated.addListener((downloadItem) => {
  if (downloadItem.url.startsWith("data:")) {
    // Don't capture blobs
    return;
  }

  // Capture the download information
  const linkInfo = {
    url: downloadItem.url,
    filename: downloadItem.filename,
    timestamp: new Date().toISOString(),
    referrer: downloadItem.referrer || 'Unknown'
  };
  
  capturedLinks.push(linkInfo);
  console.log('Captured download:', linkInfo);
  
  // Cancel the download immediately
  chrome.downloads.cancel(downloadItem.id, () => {
    console.log('Download canceled:', downloadItem.id);
    
    // Remove from downloads history to keep it clean
    chrome.downloads.erase({ id: downloadItem.id });
  });
});

function saveCapturedLinks() {
  if (capturedLinks.length === 0) return;
  
  // Create file content
  const content = capturedLinks.map(link => link.url).join('\n');
  
  const blobUrl = `data:text/plain;base64,${btoa(content)}`;
  
  // Create a download for our captured links file
  chrome.downloads.download({
    url: blobUrl,
    filename: 'captured_links.txt',
    saveAs: false
  }, null);
}

// Message handler for popup communication
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.action === 'getCapturedLinks') {
    sendResponse({ links: capturedLinks });
  } else if (request.action === 'clearLinks') {
    capturedLinks = [];
    sendResponse({ success: true });
  } else if (request.action === 'exportLinks') {
    saveCapturedLinks();
    sendResponse({ success: true });
  }
});
