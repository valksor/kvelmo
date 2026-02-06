import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import { requireConnection, withProgress } from './helpers';

export function registerProjectCommands(
  context: vscode.ExtensionContext,
  service: MehrhofProjectService
): void {
  // ============================================================================
  // Project Commands
  // ============================================================================

  // Project plan command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectPlan', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const source = await vscode.window.showInputBox({
        prompt: 'Enter source (file path, URL, or GitHub issue reference)',
        placeHolder: 'e.g., ./roadmap.md or owner/repo#123',
      });

      if (!source) {
        return;
      }

      await withProgress('Creating project plan...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['plan', source],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Project plan created');
      });
    })
  );

  // Project tasks command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectTasks', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching project tasks...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['tasks'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Project tasks retrieved');
      });
    })
  );

  // Project edit command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectEdit', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskId = await vscode.window.showInputBox({
        prompt: 'Enter task ID to edit',
        placeHolder: 'e.g., task-1',
      });

      if (!taskId) {
        return;
      }

      const field = await vscode.window.showQuickPick(
        [
          { label: 'title', description: 'Edit task title' },
          { label: 'priority', description: 'Edit task priority' },
          { label: 'status', description: 'Edit task status' },
        ],
        { placeHolder: 'Select field to edit' }
      );

      if (!field) {
        return;
      }

      const value = await vscode.window.showInputBox({
        prompt: `Enter new ${field.label}`,
        placeHolder: `New ${field.label} value`,
      });

      if (!value) {
        return;
      }

      await withProgress('Updating project task...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['edit', taskId, `--${field.label}`, value],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Task updated');
      });
    })
  );

  // Project submit command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectSubmit', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const provider = await vscode.window.showQuickPick(
        [
          { label: 'github', description: 'Submit to GitHub Issues' },
          { label: 'gitlab', description: 'Submit to GitLab Issues' },
          { label: 'linear', description: 'Submit to Linear' },
          { label: 'jira', description: 'Submit to Jira' },
        ],
        { placeHolder: 'Select provider' }
      );

      if (!provider) {
        return;
      }

      await withProgress('Submitting project tasks...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['submit', provider.label],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Tasks submitted');
      });
    })
  );

  // Project start command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectStart', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Starting next project task...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['start'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Started next task');
      });
    })
  );

  // Project sync command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectSync', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const reference = await vscode.window.showInputBox({
        prompt: 'Enter provider reference to sync from',
        placeHolder: 'e.g., owner/repo#123 or PROJECT-123',
      });

      if (!reference) {
        return;
      }

      await withProgress('Syncing project...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['sync', reference],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Project synced');
      });
    })
  );

  // ============================================================================
  // Stack Commands
  // ============================================================================

  // Stack list command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.stackList', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching stacks...', async () => {
        const response = await service.client!.executeCommand({
          command: 'stack',
          args: ['list'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Stacks retrieved');
      });
    })
  );

  // Stack rebase command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.stackRebase', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskId = await vscode.window.showInputBox({
        prompt: 'Enter task ID to rebase (leave empty to rebase all)',
        placeHolder: 'e.g., task-1 (optional)',
      });

      await withProgress('Rebasing stack...', async () => {
        const args = taskId ? ['rebase', taskId] : ['rebase'];
        const response = await service.client!.executeCommand({
          command: 'stack',
          args,
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Stack rebased');
      });
    })
  );

  // Stack sync command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.stackSync', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Syncing stacks...', async () => {
        const response = await service.client!.executeCommand({
          command: 'stack',
          args: ['sync'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Stacks synced');
      });
    })
  );

  // ============================================================================
  // Configuration Commands
  // ============================================================================

  // Config validate command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.configValidate', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Validating configuration...', async () => {
        const response = await service.client!.executeCommand({
          command: 'config',
          args: ['validate'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Configuration valid');
      });
    })
  );

  // Agents list command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.agentsList', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Listing agents...', async () => {
        const response = await service.client!.executeCommand({
          command: 'agents',
          args: ['list'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Agents retrieved');
      });
    })
  );

  // Agents explain command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.agentsExplain', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const name = await vscode.window.showInputBox({
        prompt: 'Enter agent name to explain',
        placeHolder: 'e.g., claude',
      });

      if (!name) {
        return;
      }

      await withProgress('Getting agent info...', async () => {
        const response = await service.client!.executeCommand({
          command: 'agents',
          args: ['explain', name],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Agent info retrieved');
      });
    })
  );

  // Providers list command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.providersList', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Listing providers...', async () => {
        const response = await service.client!.executeCommand({
          command: 'providers',
          args: ['list'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Providers retrieved');
      });
    })
  );

  // Providers info command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.providersInfo', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const name = await vscode.window.showInputBox({
        prompt: 'Enter provider name',
        placeHolder: 'e.g., github, jira, linear',
      });

      if (!name) {
        return;
      }

      await withProgress('Getting provider info...', async () => {
        const response = await service.client!.executeCommand({
          command: 'providers',
          args: ['info', name],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Provider info retrieved');
      });
    })
  );

  // Templates list command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.templatesList', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Listing templates...', async () => {
        const response = await service.client!.executeCommand({
          command: 'templates',
          args: ['list'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Templates retrieved');
      });
    })
  );

  // Templates show command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.templatesShow', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const name = await vscode.window.showInputBox({
        prompt: 'Enter template name',
        placeHolder: 'e.g., bug-fix, feature, refactor',
      });

      if (!name) {
        return;
      }

      await withProgress('Getting template...', async () => {
        const response = await service.client!.executeCommand({
          command: 'templates',
          args: ['show', name],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Template retrieved');
      });
    })
  );

  // Scan command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.scan', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Running security scan...', async () => {
        const response = await service.client!.executeCommand({
          command: 'scan',
          args: [],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Scan complete');
      });
    })
  );

  // Commit command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.commit', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Running commit analysis...', async () => {
        const response = await service.client!.executeCommand({
          command: 'commit',
          args: [],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Commit analysis complete');
      });
    })
  );
}
