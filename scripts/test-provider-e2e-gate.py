#!/usr/bin/env python3
"""Exercise the actual Actions shell gate with a fake Go command; never post."""
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap

root = Path(__file__).resolve().parent.parent
workflow = (root / '.github/workflows/e2e-workflow.yml').read_text()
script = textwrap.dedent(workflow.split('        run: |\n', 1)[1])

with tempfile.TemporaryDirectory(prefix='streamsignal-e2e-gate-') as directory:
    temporary = Path(directory)
    command = temporary / 'go'
    command.write_text('#!/bin/sh\nprintf "%s:%s" "$E2E_EXPECT_BLUESKY" "$E2E_EXPECT_MASTODON" > "$STREAMSIGNAL_GATE_TRACE"\n')
    command.chmod(0o700)
    for event in ('workflow_dispatch', 'workflow_call', 'push'):
        for enabled in ('false', 'true'):
            for providers in ('false', 'true'):
                trace = temporary / 'invoked'
                trace.unlink(missing_ok=True)
                shell = script
                values = {'github.event_name': event, 'inputs.enabled': enabled,
                          'inputs.run_bluesky': providers, 'inputs.run_mastodon': providers}
                for key, value in values.items():
                    shell = shell.replace('${{ ' + key + ' }}', value)
                environment = dict(os.environ, PATH=str(temporary) + os.pathsep + os.environ['PATH'],
                                   STREAMSIGNAL_GATE_TRACE=str(trace))
                subprocess.run(['bash', '-eu', '-c', shell], env=environment, check=True,
                               stdout=subprocess.DEVNULL)
                assert trace.exists() == (enabled == 'true'), (event, enabled, providers)
                if trace.exists():
                    expected = '1:1' if event == 'push' or providers == 'true' else '0:0'
                    assert trace.read_text() == expected, (event, enabled, providers)
print('Provider E2E gate: 12 non-posting scenarios passed.')
