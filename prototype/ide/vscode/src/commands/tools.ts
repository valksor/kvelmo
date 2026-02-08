import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import { requireConnection, withProgress } from './helpers';

export function registerToolCommands(
  context: vscode.ExtensionContext,
  service: MehrhofProjectService
): void {
  // ============================================================================
  // Auto Workflow
  // ============================================================================

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.auto', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const loopsStr = await vscode.window.showInputBox({
        prompt: 'Enter number of loops (0 for continuous, leave empty for default)',
        placeHolder: '0',
      });

      const loops = loopsStr ? parseInt(loopsStr, 10) : 0;

      await withProgress('Running auto workflow...', async () => {
        const response = await service.client!.auto({ loops: isNaN(loops) ? 0 : loops });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        if (response.message) {
          void vscode.window.showInformationMessage(`Mehrhof: ${response.message}`);
        }
        await service.refreshState();
      });
    })
  );

  // ============================================================================
  // Budget Commands
  // ============================================================================

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.budgetStatus', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching budget status...', async () => {
        const response = await service.client!.budgetStatus();

        if (!response.enabled) {
          void vscode.window.showInformationMessage(
            'Monthly budget is not enabled. Enable it in .mehrhof/config.yaml'
          );
          return;
        }

        const currency = response.currency ?? 'USD';
        const spent = response.spent?.toFixed(2) ?? '0.00';
        const maxCost = response.max_cost?.toFixed(2) ?? '0.00';
        const remaining = response.remaining?.toFixed(2) ?? '0.00';

        let status = `Monthly Budget: ${currency} ${spent} / ${maxCost} (${remaining} remaining)`;

        if (response.limit_hit) {
          status += ' - LIMIT REACHED';
        } else if (response.warned) {
          status += ' - Warning threshold reached';
        }

        void vscode.window.showInformationMessage(status);
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.budgetReset', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const confirm = await vscode.window.showWarningMessage(
        'Reset the monthly budget spending counter?',
        { modal: true },
        'Reset'
      );

      if (confirm !== 'Reset') {
        return;
      }

      await withProgress('Resetting budget...', async () => {
        const response = await service.client!.budgetReset();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Budget reset successfully');
      });
    })
  );

  // ============================================================================
  // Simplify Command
  // ============================================================================

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.simplify', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const path = await vscode.window.showInputBox({
        prompt: 'Enter file or directory path to simplify (leave empty for current task)',
        placeHolder: 'src/file.ts',
      });

      const instructions = await vscode.window.showInputBox({
        prompt: 'Enter simplification instructions (optional)',
        placeHolder: 'Remove unused imports, simplify conditionals',
      });

      await withProgress('Simplifying code...', async () => {
        const response = await service.client!.simplify({
          path: path || undefined,
          instructions: instructions || undefined,
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(
          response.message ?? 'Code simplification completed'
        );
        await service.refreshState();
      });
    })
  );

  // ============================================================================
  // Label Commands
  // ============================================================================

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.labelList', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching labels...', async () => {
        const response = await service.client!.labelsList();

        if (response.labels.length === 0) {
          void vscode.window.showInformationMessage('No labels on current task');
          return;
        }

        const items = response.labels.map((label) => ({ label }));
        await vscode.window.showQuickPick(items, {
          placeHolder: `Task Labels (${response.count})`,
          canPickMany: false,
        });
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.labelAdd', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const label = await vscode.window.showInputBox({
        prompt: 'Enter label to add',
        placeHolder: 'bug, feature, urgent',
      });

      if (!label) {
        return;
      }

      await withProgress('Adding label...', async () => {
        const response = await service.client!.labelsAdd(label);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? `Label '${label}' added`);
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.labelRemove', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const label = await vscode.window.showInputBox({
        prompt: 'Enter label to remove',
        placeHolder: 'bug',
      });

      if (!label) {
        return;
      }

      await withProgress('Removing label...', async () => {
        const response = await service.client!.labelsRemove(label);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? `Label '${label}' removed`);
      });
    })
  );
}
