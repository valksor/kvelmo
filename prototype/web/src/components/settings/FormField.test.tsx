import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  TextInput,
  NumberInput,
  Checkbox,
  Select,
  TextArea,
  CollapseSection,
} from './FormField'

describe('TextInput', () => {
  it('renders label and input', () => {
    const onChange = vi.fn()
    render(<TextInput label="Username" value="test" onChange={onChange} />)

    expect(screen.getByText('Username')).toBeInTheDocument()
    expect(screen.getByDisplayValue('test')).toBeInTheDocument()
  })

  it('renders hint when provided', () => {
    const onChange = vi.fn()
    render(<TextInput label="Username" value="" onChange={onChange} hint="Enter your username" />)

    expect(screen.getByText('Enter your username')).toBeInTheDocument()
  })

  it('calls onChange when typing', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(<TextInput label="Username" value="" onChange={onChange} />)

    const input = screen.getByRole('textbox')
    await user.type(input, 'a')

    expect(onChange).toHaveBeenCalledWith('a')
  })

  it('supports password type', () => {
    const onChange = vi.fn()
    render(<TextInput label="Password" value="secret" onChange={onChange} type="password" />)

    const input = document.querySelector('input[type="password"]')
    expect(input).toBeInTheDocument()
  })

  it('can be disabled', () => {
    const onChange = vi.fn()
    render(<TextInput label="Username" value="" onChange={onChange} disabled />)

    expect(screen.getByRole('textbox')).toBeDisabled()
  })
})

describe('NumberInput', () => {
  it('renders with number value', () => {
    const onChange = vi.fn()
    render(<NumberInput label="Amount" value={42} onChange={onChange} />)

    expect(screen.getByDisplayValue('42')).toBeInTheDocument()
  })

  it('respects min and max attributes', () => {
    const onChange = vi.fn()
    render(<NumberInput label="Amount" value={5} onChange={onChange} min={0} max={10} />)

    const input = screen.getByRole('spinbutton')
    expect(input).toHaveAttribute('min', '0')
    expect(input).toHaveAttribute('max', '10')
  })

  it('calls onChange with parsed number value', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(<NumberInput label="Amount" value={undefined} onChange={onChange} />)

    const input = screen.getByRole('spinbutton')
    await user.type(input, '5')

    // onChange is called with the parsed number
    expect(onChange).toHaveBeenCalledWith(5)
  })
})

describe('Checkbox', () => {
  it('renders with label', () => {
    const onChange = vi.fn()
    render(<Checkbox label="Enable feature" checked={false} onChange={onChange} />)

    expect(screen.getByText('Enable feature')).toBeInTheDocument()
  })

  it('reflects checked state', () => {
    const onChange = vi.fn()
    render(<Checkbox label="Enable feature" checked={true} onChange={onChange} />)

    expect(screen.getByRole('checkbox')).toBeChecked()
  })

  it('reflects unchecked state', () => {
    const onChange = vi.fn()
    render(<Checkbox label="Enable feature" checked={false} onChange={onChange} />)

    expect(screen.getByRole('checkbox')).not.toBeChecked()
  })

  it('calls onChange when toggled', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(<Checkbox label="Enable feature" checked={false} onChange={onChange} />)

    await user.click(screen.getByRole('checkbox'))

    expect(onChange).toHaveBeenCalledWith(true)
  })

  it('renders hint when provided', () => {
    const onChange = vi.fn()
    render(<Checkbox label="Enable" checked={false} onChange={onChange} hint="Toggle this setting" />)

    expect(screen.getByText('Toggle this setting')).toBeInTheDocument()
  })
})

describe('Select', () => {
  const options = [
    { value: 'a', label: 'Option A' },
    { value: 'b', label: 'Option B' },
  ]

  it('renders label and options', () => {
    const onChange = vi.fn()
    render(<Select label="Choose" value="" onChange={onChange} options={options} />)

    expect(screen.getByText('Choose')).toBeInTheDocument()
    expect(screen.getByRole('combobox')).toBeInTheDocument()
    expect(screen.getByText('Option A')).toBeInTheDocument()
    expect(screen.getByText('Option B')).toBeInTheDocument()
  })

  it('shows placeholder option', () => {
    const onChange = vi.fn()
    render(<Select label="Choose" value="" onChange={onChange} options={options} />)

    expect(screen.getByText('Select...')).toBeInTheDocument()
  })

  it('calls onChange when option selected', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(<Select label="Choose" value="" onChange={onChange} options={options} />)

    await user.selectOptions(screen.getByRole('combobox'), 'a')

    expect(onChange).toHaveBeenCalledWith('a')
  })

  it('can be disabled', () => {
    const onChange = vi.fn()
    render(<Select label="Choose" value="" onChange={onChange} options={options} disabled />)

    expect(screen.getByRole('combobox')).toBeDisabled()
  })
})

describe('TextArea', () => {
  it('renders with label and value', () => {
    const onChange = vi.fn()
    render(<TextArea label="Description" value="Hello world" onChange={onChange} />)

    expect(screen.getByText('Description')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Hello world')).toBeInTheDocument()
  })

  it('calls onChange when typing', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(<TextArea label="Notes" value="" onChange={onChange} />)

    await user.type(screen.getByRole('textbox'), 'a')

    expect(onChange).toHaveBeenCalledWith('a')
  })

  it('respects rows prop', () => {
    const onChange = vi.fn()
    render(<TextArea label="Notes" value="" onChange={onChange} rows={5} />)

    expect(screen.getByRole('textbox')).toHaveAttribute('rows', '5')
  })
})

describe('CollapseSection', () => {
  it('renders title button', () => {
    render(
      <CollapseSection title="Section Title">
        <p>Section content</p>
      </CollapseSection>
    )

    expect(screen.getByRole('button', { name: 'Section Title' })).toBeInTheDocument()
  })

  it('is collapsed by default', () => {
    render(
      <CollapseSection title="Section">
        <p>Content</p>
      </CollapseSection>
    )

    expect(screen.getByRole('button', { name: 'Section' })).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('Content')).not.toBeInTheDocument()
  })

  it('is open when defaultOpen is true', () => {
    render(
      <CollapseSection title="Section" defaultOpen>
        <p>Content</p>
      </CollapseSection>
    )

    expect(screen.getByRole('button', { name: 'Section' })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Content')).toBeInTheDocument()
  })

  it('toggles content visibility when clicked', async () => {
    const user = userEvent.setup()
    render(
      <CollapseSection title="Section">
        <p>Content</p>
      </CollapseSection>
    )

    const button = screen.getByRole('button', { name: 'Section' })
    await user.click(button)
    expect(button).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Content')).toBeInTheDocument()

    await user.click(button)
    expect(button).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('Content')).not.toBeInTheDocument()
  })
})
