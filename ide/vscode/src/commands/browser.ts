import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import { truncateUrl, requireConnection, withProgress } from './helpers';

export function registerBrowserCommands(
  context: vscode.ExtensionContext,
  service: MehrhofProjectService
): void {
  // Browser status command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserStatus', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Checking browser status...', async () => {
        const response = await service.client!.browserStatus();
        if (!response.connected) {
          void vscode.window.showInformationMessage(
            `Browser: Not connected${response.error ? ` (${response.error})` : ''}`
          );
          return;
        }
        const message = `Browser: Connected to ${response.host}:${response.port}\nTabs: ${response.tabs?.length ?? 0}`;
        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Browser tabs command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserTabs', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching browser tabs...', async () => {
        const response = await service.client!.browserTabs();
        if (response.count === 0) {
          void vscode.window.showInformationMessage('No browser tabs open');
          return;
        }

        const items = response.tabs.map((tab) => ({
          label: tab.title || 'Untitled',
          description: truncateUrl(tab.url, 60),
          detail: tab.id,
        }));

        await vscode.window.showQuickPick(items, {
          placeHolder: `${response.count} tab(s)`,
        });
      });
    })
  );

  // Browser go to command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserGoto', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const url = await vscode.window.showInputBox({
        prompt: 'Enter URL to navigate to',
        placeHolder: 'https://example.com',
      });

      if (!url) {
        return;
      }

      await withProgress('Opening URL...', async () => {
        const response = await service.client!.browserGoto(url);
        if (response.success && response.tab) {
          void vscode.window.showInformationMessage(
            `Opened: ${response.tab.title || response.tab.url}`
          );
        }
      });
    })
  );

  // Browser navigate command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserNavigate', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const url = await vscode.window.showInputBox({
        prompt: 'Enter URL to navigate current tab to',
        placeHolder: 'https://example.com',
      });

      if (!url) {
        return;
      }

      await withProgress('Navigating...', async () => {
        const response = await service.client!.browserNavigate(url);
        if (!response.success) {
          throw new Error('Navigation failed');
        }
        void vscode.window.showInformationMessage(response.message ?? 'Navigated');
      });
    })
  );

  // Browser reload command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserReload', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Reloading page...', async () => {
        const response = await service.client!.browserReload({});
        if (!response.success) {
          throw new Error('Reload failed');
        }
        void vscode.window.showInformationMessage(response.message ?? 'Page reloaded');
      });
    })
  );

  // Browser screenshot command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserScreenshot', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Taking screenshot...', async () => {
        const response = await service.client!.browserScreenshot({});
        if (!response.success || !response.data) {
          throw new Error('Screenshot failed');
        }

        // Show base64 image info
        const sizeKb = response.size ? Math.round(response.size / 1024) : 0;
        void vscode.window.showInformationMessage(
          `Screenshot captured: ${response.format ?? 'png'}, ${sizeKb} KB`
        );
      });
    })
  );

  // Browser click command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserClick', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const selector = await vscode.window.showInputBox({
        prompt: 'Enter CSS selector to click',
        placeHolder: '#button, .submit-btn, button[type="submit"]',
      });

      if (!selector) {
        return;
      }

      await withProgress('Clicking element...', async () => {
        const response = await service.client!.browserClick(selector);
        if (!response.success) {
          throw new Error('Click failed');
        }
        void vscode.window.showInformationMessage(`Clicked: ${response.selector ?? selector}`);
      });
    })
  );

  // Browser type command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserType', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const selector = await vscode.window.showInputBox({
        prompt: 'Enter CSS selector for input element',
        placeHolder: '#search, input[name="query"]',
      });

      if (!selector) {
        return;
      }

      const text = await vscode.window.showInputBox({
        prompt: 'Enter text to type',
        placeHolder: 'Text to type...',
      });

      if (text === undefined) {
        return;
      }

      await withProgress('Typing...', async () => {
        const response = await service.client!.browserType(selector, text);
        if (!response.success) {
          throw new Error('Type failed');
        }
        void vscode.window.showInformationMessage(`Typed into: ${response.selector ?? selector}`);
      });
    })
  );

  // Browser eval command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserEval', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const expression = await vscode.window.showInputBox({
        prompt: 'Enter JavaScript expression to evaluate',
        placeHolder: 'document.title',
      });

      if (!expression) {
        return;
      }

      await withProgress('Evaluating...', async () => {
        const response = await service.client!.browserEval(expression);
        if (!response.success) {
          throw new Error('Evaluation failed');
        }
        const resultStr = JSON.stringify(response.result, null, 2);
        void vscode.window.showInformationMessage(`Result: ${resultStr}`, { modal: true });
      });
    })
  );

  // Browser console command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserConsole', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching console logs...', async () => {
        const response = await service.client!.browserConsole({});
        if (!response.messages || response.messages.length === 0) {
          void vscode.window.showInformationMessage('No console messages');
          return;
        }

        const items = response.messages.map((msg) => ({
          label: `[${msg.level.toUpperCase()}]`,
          description: msg.text.substring(0, 100),
          detail: msg.timestamp,
        }));

        await vscode.window.showQuickPick(items, {
          placeHolder: `${response.count ?? response.messages.length} message(s)`,
        });
      });
    })
  );

  // Browser network command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserNetwork', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching network requests...', async () => {
        const response = await service.client!.browserNetwork({});
        if (!response.requests || response.requests.length === 0) {
          void vscode.window.showInformationMessage('No network requests');
          return;
        }

        const items = response.requests.map((req) => ({
          label: `${req.method} ${req.status ?? '...'}`,
          description: truncateUrl(req.url, 60),
          detail: req.timestamp,
        }));

        await vscode.window.showQuickPick(items, {
          placeHolder: `${response.count ?? response.requests.length} request(s)`,
        });
      });
    })
  );
}
