package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"muslimboard-api.novalagung.com/models"
	pkg_common "muslimboard-api.novalagung.com/pkg/common"
	pkg_http "muslimboard-api.novalagung.com/pkg/http"
	"muslimboard-api.novalagung.com/pkg/logger"
	"muslimboard-api.novalagung.com/pkg/otel"
	pkg_redis "muslimboard-api.novalagung.com/pkg/redis"
	"muslimboard-api.novalagung.com/usecase"
)

// HandleImage is handler of get image action
// currently not isued
func HandleImage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	namespace := models.Namespace("handler.HandleImage")
	log := logger.New(namespace)

	ctx, span := otel.Tracer.Start(ctx, namespace)
	defer span.End()

	imageUrl, _ := url.QueryUnescape(r.URL.Query().Get("image"))
	if imageUrl == "" {
		err := fmt.Errorf("missing image url")
		log.Error(ctx, fmt.Errorf("queryUnescape %w", err))
		pkg_http.WriteRespose(w, r, http.StatusBadRequest, nil, err)
		return
	}

	contentType, body, err := usecase.GetImage(ctx, imageUrl)
	if body != nil {
		defer body.Close()
	}
	if err != nil {
		log.Error(ctx, fmt.Errorf("getImage %w", err))
		pkg_http.WriteRespose(w, r, http.StatusBadRequest, nil, err)
		return
	}

	pkg_http.RenderCacheHeader(w, r)
	w.Header().Set("Content-type", contentType)

	_, err = io.Copy(w, body)
	if err != nil {
		log.Error(ctx, fmt.Errorf("io.Copy %w", err))
		pkg_http.WriteRespose(w, r, http.StatusBadRequest, nil, err)
		return
	}
}

// HandleShalatScheduleByCoordinate is handler of get shalat schedule by coordinate action
func HandleShalatScheduleByCoordinate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	namespace := models.Namespace("handler.HandleShalatScheduleByCoordinate")
	log := logger.New(namespace)

	ctx, span := otel.Tracer.Start(ctx, namespace)
	defer span.End()

	// check cache
	cacheKey := r.URL.String()
	cachedRes, err := pkg_redis.NewRedis().Get(ctx, cacheKey).Result()
	if err == nil {
		cachedResMap, err := pkg_common.ConvertToMap(cachedRes)
		if len(cachedResMap) > 0 && err == nil {
			// prolong the cache expiration date
			pkg_redis.NewRedis().Set(ctx, cacheKey, cachedRes, models.RedisKeepAliveDuration).Err()
			log.Debug(ctx, "load from cache", cacheKey)
			pkg_http.WriteRespose(w, r, http.StatusOK, cachedResMap, nil)
			return
		}
	}

	// parse params
	method := r.URL.Query().Get("method")
	if method == "" {
		method = "1"
	}
	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")
	latitude := r.URL.Query().Get("latitude")
	longitude := r.URL.Query().Get("longitude")

	// if lat long is invalid, then simply return true
	latInt, _ := strconv.ParseFloat(latitude, 64)
	lonInt, _ := strconv.ParseFloat(longitude, 64)
	if latInt == 0 && lonInt == 0 {
		pkg_http.WriteRespose(w, r, http.StatusOK, true, nil)
		return
	}

	// get data
	res, err := usecase.GetShalatScheduleByCoordinate(ctx, method, latitude, longitude, month, year)
	if err != nil {
		log.Error(ctx, fmt.Errorf("getShalatScheduleByCoordinate %w", err))
		pkg_http.WriteRespose(w, r, http.StatusInternalServerError, nil, err)
		return
	}

	// cache data
	if schedulesRaw := res["schedules"]; schedulesRaw != nil {
		if schedules := schedulesRaw.([]map[string]interface{}); len(schedules) > 0 {
			log.Debug(ctx, "set cache", cacheKey)
			pkg_redis.NewRedis().Set(ctx, cacheKey, pkg_common.ConvertToJson(res), models.RedisKeepAliveDuration).Err()
		}
	}

	pkg_http.WriteRespose(w, r, http.StatusOK, res, nil)
}

// HandleShalatScheduleByLocation is handler of get shalat schedule by location action
// for now, immediately use aladhan.com api coz kemenag backend still under development
func HandleShalatScheduleByLocation(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	namespace := models.Namespace("handler.HandleShalatScheduleByLocation")
	log := logger.New(namespace)

	ctx, span := otel.Tracer.Start(ctx, namespace)
	defer span.End()

	// check cache
	cacheKey := r.URL.String()
	cachedRes, err := pkg_redis.NewRedis().Get(ctx, cacheKey).Result()
	if err == nil {
		cachedResMap, err := pkg_common.ConvertToMap(cachedRes)
		if len(cachedResMap) > 0 && err == nil {
			// prolong the cache expiration date
			pkg_redis.NewRedis().Set(ctx, cacheKey, cachedRes, models.RedisKeepAliveDuration).Err()
			log.Debug(ctx, "load from cache", cacheKey)
			pkg_http.WriteRespose(w, r, http.StatusOK, cachedResMap, nil)
			return
		}
	}

	// parse params
	method := r.URL.Query().Get("method")
	if method == "" {
		method = "1"
	}
	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")
	province := r.URL.Query().Get("province")
	city := r.URL.Query().Get("city")

	// get data
	res, err := usecase.GetShalatScheduleByLocation(ctx, method, province, city, month, year)
	if err != nil {
		log.Error(ctx, fmt.Errorf("getShalatScheduleByCoordinate %w", err))
		pkg_http.WriteRespose(w, r, http.StatusInternalServerError, nil, err)
		return
	}

	// cache data
	if schedulesRaw := res["schedules"]; schedulesRaw != nil {
		if schedules := schedulesRaw.([]map[string]interface{}); len(schedules) > 0 {
			log.Debug(ctx, "set cache", cacheKey)
			pkg_redis.NewRedis().Set(ctx, cacheKey, pkg_common.ConvertToJson(res), models.RedisKeepAliveDuration).Err()
		}
	}

	pkg_http.WriteRespose(w, r, http.StatusOK, res, nil)
}
