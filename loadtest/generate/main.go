package main

import (
	"bannersrv/internal/app/config"
	bp "bannersrv/internal/banner/repository/postgres"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/logger"
	"encoding/json"
	"flag"
	"github.com/go-co-op/gocron/v2"
	"github.com/jmoiron/sqlx"
	"github.com/tidwall/randjson"
	"log"
	"os"
)

type CorrectPair struct {
	BannerId  types.Id   `json:"banner_id"`
	FeatureId types.Id   `json:"feature_id"`
	TagIds    []types.Id `json:"tag_ids"`
}

func main() {
	var featuresCount, tagsCount, countBanners uint64
	var configPath string

	flag.StringVar(&configPath, "config", "./config/localhost-config.yaml", "путь к конфигу подключения")
	flag.Uint64Var(&featuresCount, "features", 1000, "число фичей")
	flag.Uint64Var(&tagsCount, "tags", 1000, "число тэгов")
	flag.Uint64Var(&countBanners, "banners", 8000, "число баннеров")
	flag.Parse()

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Postgres
	pg, err := sqlx.Open("postgres", cfg.Postgres.URL)
	if err != nil {
		log.Fatalf("postgres.New: %s", err)
	}
	defer pg.Close()

	if err := pg.Ping(); err != nil {
		log.Fatalf("can't check connection to sql with error %s", err)
	}

	// Cron
	cronScheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("start cronScheduler error: %s", err)
	}

	// Repository
	bannerRepository, err := bp.NewBannerRepository(pg, cronScheduler, &logger.EmptyLogger{})
	if err != nil {
		log.Fatalf("[initialize BannerRepository error: %s", err)
	}

	featureIds := make([]types.Id, featuresCount)
	tagsIds := make([]types.Id, tagsCount)

	for i := uint64(0); i < featuresCount; i++ {
		featureIds[i] = types.Id(i + 1)
	}

	for i := uint64(0); i < tagsCount; i++ {
		tagsIds[i] = types.Id(i + 1)
	}

	countTagsInBanner := tagsCount / (countBanners / featuresCount)

	result := make([]CorrectPair, 0)

	for i := uint64(0); i < featuresCount; i++ {
		for j := uint64(0); j < tagsCount; j += countTagsInBanner {
			tags := tagsIds[j:min(j+countTagsInBanner, tagsCount)]
			featureId := featureIds[i]
			banner := randjson.Make(12, nil)

			createdId, err := bannerRepository.CreateBanner(featureId, tags, types.Content(banner), true)
			if err != nil {
				log.Fatal(err)
			}

			result = append(result, CorrectPair{
				BannerId:  createdId,
				FeatureId: featureId,
				TagIds:    tags,
			})
		}
	}

	file, err := os.OpenFile("./info.json", os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Fatal(err)
	}

	res, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}

	_, err = file.Write(res)
	if err != nil {
		log.Fatal(err)
	}

	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
}
