from aiogram.types import CallbackQuery, InputMediaPhoto, Message


async def edit_or_replace_comic(
    call: CallbackQuery,
    *,
    url: str,
    caption: str,
    text: str,
    reply_markup,
) -> Message:

    #Если тип сообщения совпадает (photo->photo или text->text) - редактируем
    #Если тип меняется - удаляем старое и отправляем новое, возвращаем новое сообщение

    desired_photo = bool(url)
    current_is_photo = bool(call.message.photo)

    # хотим фото
    if desired_photo:
        if current_is_photo:
            try:
                await call.message.edit_media(
                    media=InputMediaPhoto(media=url, caption=caption),
                    reply_markup=reply_markup,
                )
                return call.message
            except Exception:
                pass  # fallback на replace

        # replace: уберём клаву у старого и отправим новое
        try:
            await call.message.edit_reply_markup(reply_markup=None)
        except Exception:
            pass
        try:
            await call.message.delete()
        except Exception:
            pass

        return await call.message.answer_photo(photo=url, caption=caption, reply_markup=reply_markup)

    # хотим текст
    if not current_is_photo:
        try:
            await call.message.edit_text(text, reply_markup=reply_markup)
            return call.message
        except Exception:
            pass  # fallback на replace

    try:
        await call.message.edit_reply_markup(reply_markup=None)
    except Exception:
        pass
    try:
        await call.message.delete()
    except Exception:
        pass

    return await call.message.answer(text, reply_markup=reply_markup)
