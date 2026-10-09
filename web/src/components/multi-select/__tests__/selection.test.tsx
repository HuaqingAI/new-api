/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createRef, useState } from 'react'
import { describe, expect, it } from 'vitest'

import { MultiSelect } from '@/components/multi-select'

const options = [
  { value: 'a', label: 'a' },
  { value: 'c', label: 'c' },
]

function Fixture(props: {
  initialSelected?: string[]
  allowCreate?: boolean
  disabled?: boolean
}) {
  const [selected, setSelected] = useState(props.initialSelected ?? [])

  return (
    <>
      <MultiSelect
        options={options}
        selected={selected}
        onChange={setSelected}
        allowCreate={props.allowCreate ?? true}
        disabled={props.disabled}
      />
      <output aria-label='Selected values'>{selected.join(',')}</output>
    </>
  )
}

describe('multi-select batch paste', () => {
  it('forwards field accessibility and an input ref so invalid selections can receive focus', () => {
    const inputRef = createRef<HTMLInputElement>()
    render(
      <>
        <label htmlFor='models'>Models</label>
        <MultiSelect
          id='models'
          ref={inputRef}
          options={options}
          selected={[]}
          onChange={() => undefined}
          aria-invalid
          aria-describedby='models-error'
        />
        <p id='models-error'>Invalid model ID</p>
      </>
    )
    const input = screen.getByRole('combobox')
    inputRef.current?.focus()
    expect(input).toHaveFocus()
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input).toHaveAccessibleDescription('Invalid model ID')
  })
  it.each(['a,b,c', 'a，b，c', 'a\nb\nc', 'a\r\nb\r\nc'])(
    'adds every pasted value including the final item without a trailing separator: %j',
    async (text) => {
      const user = userEvent.setup()
      render(<Fixture />)
      const input = screen.getByRole('combobox')

      await user.click(input)
      await user.paste(text)

      expect(screen.getByLabelText('Selected values')).toHaveTextContent(
        /^a,b,c$/
      )
      expect(input).toHaveValue('')
      expect(input).toHaveFocus()
    }
  )

  it('preserves existing selections and ignores duplicate, blank, and trailing entries', async () => {
    const user = userEvent.setup()
    render(<Fixture initialSelected={['existing', 'a']} />)
    const input = screen.getByRole('combobox')

    await user.click(input)
    await user.paste(' a,, b，b\r\nc, ')

    expect(screen.getByLabelText('Selected values')).toHaveTextContent(
      /^existing,a,b,c$/
    )
    expect(input).toHaveValue('')
  })

  it('replaces the selected draft text when pasting a batch and preserves surrounding text', async () => {
    const user = userEvent.setup()
    render(<Fixture />)
    const input = screen.getByRole<HTMLInputElement>('combobox')

    await user.type(input, 'model-old-end')
    input.setSelectionRange(6, 9)
    await user.paste('a,b')

    expect(screen.getByLabelText('Selected values')).toHaveTextContent(
      /^model-a,b-end$/
    )
    expect(input).toHaveValue('')
  })

  it('keeps the last manually typed value as a draft until a separator is typed', async () => {
    const user = userEvent.setup()
    render(<Fixture />)
    const input = screen.getByRole('combobox')

    await user.type(input, 'a,b,c')

    expect(screen.getByLabelText('Selected values')).toHaveTextContent(/^a,b$/)
    expect(input).toHaveValue('c')

    await user.type(input, ',')

    expect(screen.getByLabelText('Selected values')).toHaveTextContent(
      /^a,b,c$/
    )
    expect(input).toHaveValue('')
  })

  it('keeps a single pasted value available for search and explicit selection', async () => {
    const user = userEvent.setup()
    render(<Fixture />)
    const input = screen.getByRole('combobox')

    await user.click(input)
    await user.paste('c')

    expect(screen.getByLabelText('Selected values')).toBeEmptyDOMElement()
    expect(input).toHaveValue('c')

    await user.click(screen.getByRole('option', { name: 'c' }))

    expect(screen.getByLabelText('Selected values')).toHaveTextContent(/^c$/)
    expect(input).toHaveValue('')
  })

  it('leaves pasted text as search input when custom creation is disabled', async () => {
    const user = userEvent.setup()
    render(<Fixture allowCreate={false} />)
    const input = screen.getByRole('combobox')

    await user.click(input)
    await user.paste('a,b,c')

    expect(screen.getByLabelText('Selected values')).toBeEmptyDOMElement()
    expect(input).toHaveValue('a,b,c')
  })

  it('does not add pasted values when the control is disabled', async () => {
    const user = userEvent.setup()
    render(<Fixture disabled />)
    const input = screen.getByRole('combobox')

    await user.click(input)
    await user.paste('a,b,c')

    expect(input).toBeDisabled()
    expect(screen.getByLabelText('Selected values')).toBeEmptyDOMElement()
    expect(input).toHaveValue('')
  })
})
