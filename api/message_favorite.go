package api

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"sealchat/protocol"
	"sealchat/service"
)

const messageFavoriteWorldChangedEvent = "message.favorite.world.changed"

func broadcastMessageFavoriteWorldChanged(worldID string) {
	ctx := &ChatContext{UserId2ConnInfo: userId2ConnInfoGlobal}
	ctx.BroadcastEventInWorld(worldID, &protocol.Event{
		Type: messageFavoriteWorldChangedEvent,
		Argv: &protocol.Argv{Options: map[string]interface{}{"worldId": worldID}},
	})
}

// BindMessageFavoriteRoutes is called after SignCheckMiddleware, just like the
// shortcut-rule routes. The guard also makes standalone route binding safe.
func BindMessageFavoriteRoutes(group fiber.Router) {
	service.SetMessageFavoriteWorldChangedCallback(broadcastMessageFavoriteWorldChanged)
	auth := func(c *fiber.Ctx) error {
		if user := getCurUser(c); user == nil || user.ID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "请先登录"})
		}
		return c.Next()
	}
	favorites := group.Group("/message-favorites", auth)
	favorites.Get("", messageFavoriteListHandler)
	favorites.Post("/batch", messageFavoriteBatchHandler)
	favorites.Post("/batch-delete", messageFavoriteBatchDeleteHandler)
	favorites.Delete("/:id", messageFavoriteDeleteHandler)
	favorites.Post("/:id/move", messageFavoriteMoveHandler)
	favorites.Put("/:id/tags", messageFavoriteTagsHandler)
	tags := group.Group("/message-favorite-tags", auth)
	tags.Get("", messageFavoriteTagListHandler)
	tags.Post("", messageFavoriteTagCreateHandler)
	tags.Patch("/:id", messageFavoriteTagRenameHandler)
	tags.Delete("/:id", messageFavoriteTagDeleteHandler)
}

func messageFavoriteError(c *fiber.Ctx, err error) error {
	status, message := fiber.StatusInternalServerError, "消息收藏操作失败"
	switch {
	case errors.Is(err, service.ErrMessageFavoriteInvalid):
		status, message = fiber.StatusBadRequest, "收藏参数无效"
	case errors.Is(err, service.ErrMessageFavoriteDenied):
		status, message = fiber.StatusForbidden, "无权操作该收藏空间"
	case errors.Is(err, service.ErrMessageFavoriteNotFound):
		status, message = fiber.StatusNotFound, "收藏、标签或世界不存在"
	case errors.Is(err, service.ErrMessageFavoriteConflict):
		status, message = fiber.StatusConflict, "标签重名或排序邻居已变更"
	}
	return c.Status(status).JSON(fiber.Map{"message": message})
}

func messageFavoriteQueryScope(c *fiber.Ctx) service.MessageFavoriteScope {
	return service.MessageFavoriteScope{Scope: c.Query("scope"), WorldID: c.Query("worldId")}
}

func messageFavoriteListHandler(c *fiber.Ctx) error {
	opts := service.MessageFavoriteListOptions{MessageFavoriteScope: messageFavoriteQueryScope(c), Query: c.Query("q"), Cursor: c.Query("cursor")}
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return messageFavoriteError(c, service.ErrMessageFavoriteInvalid)
		}
		opts.Limit = limit
	}
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		if string(key) == "tagIds" || string(key) == "tagIds[]" {
			opts.TagIDs = append(opts.TagIDs, strings.Split(string(value), ",")...)
		}
	})
	result, err := service.MessageFavoriteList(getCurUser(c).ID, opts)
	if err != nil {
		return messageFavoriteError(c, err)
	}
	return c.JSON(result)
}

func messageFavoriteBatchHandler(c *fiber.Ctx) error {
	var input service.MessageFavoriteBatchInput
	if err := c.BodyParser(&input); err != nil {
		return messageFavoriteError(c, service.ErrMessageFavoriteInvalid)
	}
	result, err := service.MessageFavoriteBatchCreate(getCurUser(c).ID, input)
	if err != nil {
		return messageFavoriteError(c, err)
	}
	return c.JSON(result)
}

func messageFavoriteBatchDeleteHandler(c *fiber.Ctx) error {
	var input service.MessageFavoriteBatchDeleteInput
	if err := c.BodyParser(&input); err != nil {
		return messageFavoriteError(c, service.ErrMessageFavoriteInvalid)
	}
	result, err := service.MessageFavoriteBatchDelete(getCurUser(c).ID, input)
	if err != nil {
		return messageFavoriteError(c, err)
	}
	return c.JSON(result)
}

func messageFavoriteDeleteHandler(c *fiber.Ctx) error {
	if err := service.MessageFavoriteDelete(getCurUser(c).ID, c.Params("id")); err != nil {
		return messageFavoriteError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func messageFavoriteMoveHandler(c *fiber.Ctx) error {
	var input service.MessageFavoriteMoveInput
	if err := c.BodyParser(&input); err != nil {
		return messageFavoriteError(c, service.ErrMessageFavoriteInvalid)
	}
	item, err := service.MessageFavoriteMove(getCurUser(c).ID, c.Params("id"), input)
	if err != nil {
		return messageFavoriteError(c, err)
	}
	return c.JSON(fiber.Map{"item": item})
}

func messageFavoriteTagsHandler(c *fiber.Ctx) error {
	var input struct {
		TagIDs *[]string `json:"tagIds"`
	}
	if err := c.BodyParser(&input); err != nil || input.TagIDs == nil {
		return messageFavoriteError(c, service.ErrMessageFavoriteInvalid)
	}
	tags, err := service.MessageFavoriteReplaceTags(getCurUser(c).ID, c.Params("id"), *input.TagIDs)
	if err != nil {
		return messageFavoriteError(c, err)
	}
	return c.JSON(fiber.Map{"tags": tags})
}

func messageFavoriteTagListHandler(c *fiber.Ctx) error {
	result, err := service.MessageFavoriteTagList(getCurUser(c).ID, messageFavoriteQueryScope(c))
	if err != nil {
		return messageFavoriteError(c, err)
	}
	return c.JSON(result)
}

func messageFavoriteTagCreateHandler(c *fiber.Ctx) error {
	var input service.MessageFavoriteTagInput
	if err := c.BodyParser(&input); err != nil {
		return messageFavoriteError(c, service.ErrMessageFavoriteInvalid)
	}
	item, err := service.MessageFavoriteTagCreate(getCurUser(c).ID, input)
	if err != nil {
		return messageFavoriteError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"item": item})
}

func messageFavoriteTagRenameHandler(c *fiber.Ctx) error {
	var input struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&input); err != nil {
		return messageFavoriteError(c, service.ErrMessageFavoriteInvalid)
	}
	item, err := service.MessageFavoriteTagRename(getCurUser(c).ID, c.Params("id"), input.Name)
	if err != nil {
		return messageFavoriteError(c, err)
	}
	return c.JSON(fiber.Map{"item": item})
}

func messageFavoriteTagDeleteHandler(c *fiber.Ctx) error {
	if err := service.MessageFavoriteTagDelete(getCurUser(c).ID, c.Params("id")); err != nil {
		return messageFavoriteError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}
