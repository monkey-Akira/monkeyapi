import { Ban } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import bananaCursorUrl from '@/assets/banana-cursor.svg'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useMediaQuery } from '@/hooks/use-media-query'
import {
  readCursorPreference,
  type CursorStyle,
  writeCursorPreference,
} from '@/lib/cursor-preference'

export function BananaCursorToggle() {
  const { t } = useTranslation()
  const hasFinePointer = useMediaQuery('(hover: hover) and (pointer: fine)')
  const [cursorStyle, setCursorStyle] = useState<CursorStyle>(() =>
    readCursorPreference()
  )
  const bananaEnabled = cursorStyle === 'banana'
  const label = bananaEnabled
    ? t('Restore the standard cursor')
    : t('Use the banana cursor')

  useEffect(() => {
    document.documentElement.dataset.cursorStyle = bananaEnabled
      ? 'banana'
      : 'default'
  }, [bananaEnabled])

  if (!hasFinePointer) return null

  const toggleCursor = () => {
    const nextStyle: CursorStyle = bananaEnabled ? 'default' : 'banana'
    setCursorStyle(nextStyle)
    writeCursorPreference(nextStyle)
  }

  return (
    <div className='banana-cursor-toggle fixed right-5 bottom-5 z-30'>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type='button'
              variant='outline'
              size='icon-lg'
              aria-label={label}
              aria-pressed={bananaEnabled}
              className='bg-background/95 hover:bg-background relative rounded-full border shadow-lg backdrop-blur'
              onClick={toggleCursor}
            />
          }
        >
          <img
            src={bananaCursorUrl}
            alt=''
            aria-hidden='true'
            className='size-6'
            draggable='false'
          />
          {!bananaEnabled && (
            <Ban
              className='text-destructive absolute size-7 stroke-[2.75]'
              aria-hidden='true'
            />
          )}
        </TooltipTrigger>
        <TooltipContent side='left'>
          <p>{label}</p>
        </TooltipContent>
      </Tooltip>
    </div>
  )
}
