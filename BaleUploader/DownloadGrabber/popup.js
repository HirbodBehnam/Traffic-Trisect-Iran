document.addEventListener('DOMContentLoaded', function() {
  const statusText = document.getElementById('statusText');
  const linksList = document.getElementById('linksList');
  const exportBtn = document.getElementById('exportBtn');
  const clearBtn = document.getElementById('clearBtn');
  
  // Load captured links on popup open
  loadCapturedLinks();
  
  function loadCapturedLinks() {
    chrome.runtime.sendMessage({ action: 'getCapturedLinks' }, (response) => {
      const links = response.links || [];
      updateStatus(links.length);
      displayLinks(links);
    });
  }
  
  function updateStatus(count) {
    statusText.textContent = `${count} links captured`;
  }
  
  function displayLinks(links) {
    if (links.length === 0) {
      linksList.innerHTML = '<div>No links captured yet</div>';
      return;
    }
    
    linksList.innerHTML = links.map(link => `
      <div class="link-item">
        <strong>${link.filename}</strong><br>
        <small>${link.url}</small><br>
        <small style="color: #666;">${new Date(link.timestamp).toLocaleString()}</small>
      </div>
    `).join('');
  }
  
  // Export button handler
  exportBtn.addEventListener('click', function() {
    chrome.runtime.sendMessage({ action: 'exportLinks' }, (response) => {
      if (response.success) {
        statusText.textContent = 'Links exported to Downloads folder!';
        setTimeout(() => {
          loadCapturedLinks();
        }, 2000);
      }
    });
  });
  
  // Clear button handler
  clearBtn.addEventListener('click', function() {
    if (confirm('Are you sure you want to clear all captured links?')) {
      chrome.runtime.sendMessage({ action: 'clearLinks' }, (response) => {
        if (response.success) {
          loadCapturedLinks();
        }
      });
    }
  });
});